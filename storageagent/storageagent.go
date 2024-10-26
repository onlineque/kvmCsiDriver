package main

import (
	"context"
	"fmt"
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
	k := kvm.Kvm{}
	_, err := k.CreateVolume(fmt.Sprintf("/var/lib/libvirt/images/%s.qcow2", req.ImageId), req.Size)
	if err != nil {
		return nil, fmt.Errorf("error while creating the QCOW2 image (%s) for the volume: %s", fmt.Sprintf("/images/%s.qcow2", req.ImageId), err)
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
