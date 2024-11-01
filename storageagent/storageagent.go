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

const QCOWImagePath = "/var/lib/libvirt/images/%s.qcow2"

type server struct {
	sa.UnimplementedStorageAgentServer
}

func (s *server) CreateImage(_ context.Context, req *sa.ImageRequest) (*sa.Image, error) {
	imageName := fmt.Sprintf(QCOWImagePath, req.ImageId)
	k := kvm.Kvm{}
	err := k.CreateVolume(imageName, req.Size)
	if err != nil {
		return nil, fmt.Errorf("error while creating the QCOW2 image (%s) for the volume: %w", imageName, err)
	}

	log.Printf("volume %s.qcow2 created", req.ImageId)
	return &sa.Image{
		Success: true,
		ImageId: req.ImageId,
	}, nil
}

func (s *server) DeleteImage(_ context.Context, req *sa.ImageRequest) (*sa.Image, error) {
	err := os.Remove(fmt.Sprintf(QCOWImagePath, req.ImageId))
	if err != nil {
		return nil, err
	}

	log.Printf("volume %s.qcow2 deleted", req.ImageId)
	return &sa.Image{
		Success: true,
		ImageId: req.ImageId,
	}, nil
}

func (s *server) AttachVolume(_ context.Context, req *sa.VolumeRequest) (*sa.Volume, error) {
	imageID := req.ImageId
	imageName := fmt.Sprintf(QCOWImagePath, imageID)
	targetPath := req.TargetPath
	domainName := req.DomainName

	log.Printf("mounting %s on %s:%s ...", imageName, domainName, targetPath)

	k := kvm.Kvm{
		URI: string(libvirt.QEMUSystem),
	}

	err := k.Connect()
	if err != nil {
		return nil, err
	}
	defer k.Disconnect()

	nextDeviceName, err := k.FindNextUsableDeviceName(domainName)
	if err != nil {
		return nil, fmt.Errorf("error looking up next free device name: %w", err)
	}

	err = k.AttachVolumeToDomain(domainName, imageName, nextDeviceName)
	if err != nil {
		return nil, err
	}
	log.Printf("successfully attached volume %s to domain %s", imageName, domainName)

	return &sa.Volume{
		ImageId: imageID,
		Success: true,
		Device:  nextDeviceName,
	}, nil
}

func (s *server) DetachVolume(_ context.Context, req *sa.VolumeRequest) (*sa.Volume, error) {
	imageID := req.ImageId
	imageName := fmt.Sprintf(QCOWImagePath, imageID)
	targetPath := req.TargetPath
	domainName := req.DomainName

	log.Printf("unmounting %s from %s:%s ...", imageName, domainName, targetPath)

	k := kvm.Kvm{
		URI: string(libvirt.QEMUSystem),
	}

	err := k.Connect()
	if err != nil {
		return nil, err
	}
	defer k.Disconnect()

	deviceName, err := k.GetDeviceNameBySource(domainName, imageName)
	if err != nil {
		return nil, fmt.Errorf("error getting the device name for the image: %w", err)
	}
	err = k.DetachVolumeFromDomain(domainName, imageName, deviceName)
	if err != nil {
		return nil, err
	}
	log.Printf("successfully detached volume %s from domain %s", imageName, domainName)

	return &sa.Volume{
		ImageId: imageID,
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
