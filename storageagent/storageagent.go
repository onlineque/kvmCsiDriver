package main

import (
	"context"
	"fmt"
	"github.com/digitalocean/go-libvirt"
	"github.com/onlineque/kvmCsiDriver/pkg/kvm"
	sa "github.com/onlineque/kvmCsiDriver/storageagent_proto"
	"google.golang.org/grpc"
	"log"
	"net"
	"os"
)

type server struct {
	sa.UnimplementedStorageAgentServer
}

func (s *server) CreateImage(ctx context.Context, req *sa.ImageRequest) (*sa.Image, error) {
	imageName := fmt.Sprintf("/var/lib/libvirt/images/%s.qcow2", req.ImageId)
	k := kvm.Kvm{}
	err := k.CreateVolume(imageName, req.Size)
	if err != nil {
		return nil, fmt.Errorf("error while creating the QCOW2 image (%s) for the volume: %s", imageName, err)
	}

	log.Printf("volume %s.qcow2 created", req.ImageId)
	return &sa.Image{
		Success: true,
		ImageId: req.ImageId,
	}, nil
}

func (s *server) DeleteImage(ctx context.Context, req *sa.ImageRequest) (*sa.Image, error) {
	err := os.Remove(fmt.Sprintf("/var/lib/libvirt/images/%s.qcow2", req.ImageId))
	if err != nil {
		return nil, err
	}

	log.Printf("volume %s.qcow2 deleted", req.ImageId)
	return &sa.Image{
		Success: true,
		ImageId: req.ImageId,
	}, nil
}

func (s *server) AttachVolume(ctx context.Context, req *sa.VolumeRequest) (*sa.Volume, error) {
	imageId := req.ImageId
	imageName := fmt.Sprintf("/var/lib/libvirt/images/%s.qcow2", imageId)
	targetPath := req.TargetPath
	domainName := req.DomainName

	log.Printf("mounting %s on %s:%s ...", imageName, domainName, targetPath)

	k := kvm.Kvm{
		Uri: string(libvirt.QEMUSystem),
	}

	err := k.Connect()
	if err != nil {
		return nil, err
	}
	defer k.Disconnect()

	nextDeviceName, err := k.FindNextUsableDeviceName(domainName)
	if err != nil {
		return nil, fmt.Errorf("error looking up next free device name: %s", err)
	}

	err = k.AttachVolumeToDomain(domainName, imageName, nextDeviceName)
	if err != nil {
		return nil, err
	}
	log.Printf("successfully attached volume %s to domain %s", imageName, domainName)

	return &sa.Volume{
		ImageId: imageId,
		Success: true,
		Device:  nextDeviceName,
	}, nil
}

func (s *server) DetachVolume(ctx context.Context, req *sa.VolumeRequest) (*sa.Volume, error) {
	imageId := req.ImageId
	imageName := fmt.Sprintf("/var/lib/libvirt/images/%s.qcow2", imageId)
	targetPath := req.TargetPath
	domainName := req.DomainName

	log.Printf("unmounting %s from %s:%s ...", imageName, domainName, targetPath)

	k := kvm.Kvm{
		Uri: string(libvirt.QEMUSystem),
	}

	err := k.Connect()
	if err != nil {
		return nil, err
	}
	defer k.Disconnect()

	deviceName, err := k.GetDeviceNameBySource(domainName, imageName)
	if err != nil {
		return nil, fmt.Errorf("error getting the device name for the image: %s", err)
	}
	err = k.DetachVolumeFromDomain(domainName, imageName, deviceName)
	if err != nil {
		return nil, err
	}
	log.Printf("successfully detached volume %s from domain %s", imageName, domainName)

	return &sa.Volume{
		ImageId: imageId,
		Success: true,
	}, nil
}

func main() {
	ctx := context.TODO()

	listener, err := net.Listen("tcp", ":7003")
	if err != nil {
		log.Fatalf("failed to listen: %v", err)
	}

	defer listener.Close()

	srv := grpc.NewServer()
	sa.RegisterStorageAgentServer(srv, &server{})

	go func() {
		err = srv.Serve(listener)
		if err != nil {
			log.Fatal(err)
		}
	}()

	log.Print("KVM CSI Driver StorageAgent has been started")

	<-ctx.Done()
	srv.GracefulStop()
	log.Print("KVM CSI Driver StorageAgent has been stopped")
}
