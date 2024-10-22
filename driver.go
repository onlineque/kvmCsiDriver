package main

import (
	"context"
	csi "github.com/onlineque/kvmCsiDriver/csi_proto"
	"google.golang.org/grpc"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/status"
	wrappers "google.golang.org/protobuf/types/known/wrapperspb"
	"log"
	"net"
	"os"
)

type nodeServer struct {
	csi.UnimplementedNodeServer
}

// the following link describes the minimum CSI driver must implement:
// https://kubernetes-csi.github.io/docs/developing.html

type identityServer struct {
	name    string
	version string
	csi.UnimplementedIdentityServer
}

func newIdentityServer(name, version string) *identityServer {
	return &identityServer{
		name:    name,
		version: version,
	}
}

func (ids *identityServer) GetPluginInfo(ctx context.Context, req *csi.GetPluginInfoRequest) (*csi.GetPluginInfoResponse, error) {
	if ids.name == "" {
		return nil, status.Error(codes.Unavailable, "driver name not configured")
	}
	if ids.version == "" {
		return nil, status.Error(codes.Unavailable, "driver version not configured")
	}
	return &csi.GetPluginInfoResponse{
		Name:          ids.name,
		VendorVersion: ids.version,
	}, nil
}

func (ids *identityServer) GetPluginCapabilities(ctx context.Context, req *csi.GetPluginCapabilitiesRequest) (*csi.GetPluginCapabilitiesResponse, error) {
	return &csi.GetPluginCapabilitiesResponse{
		Capabilities: []*csi.PluginCapability{},
	}, nil
}

func (ids *identityServer) Probe(ctx context.Context, req *csi.ProbeRequest) (*csi.ProbeResponse, error) {
	return &csi.ProbeResponse{Ready: &wrappers.BoolValue{Value: true}}, nil
}

func (ns *nodeServer) NodePublishVolume(ctx context.Context, req *csi.NodePublishVolumeRequest) (*csi.NodePublishVolumeResponse, error) {
	// mounting the volume should be here
	return &csi.NodePublishVolumeResponse{}, nil
}

func (ns *nodeServer) NodeUnpublishVolume(ctx context.Context, req *csi.NodeUnpublishVolumeRequest) (*csi.NodeUnpublishVolumeResponse, error) {
	// unmounting  the volume should be here
	return &csi.NodeUnpublishVolumeResponse{}, nil
}

func (ns *nodeServer) NodeGetCapabilities(ctx context.Context, req *csi.NodeGetCapabilitiesRequest) (*csi.NodeGetCapabilitiesResponse, error) {
	caps := []*csi.NodeServiceCapability{}

	return &csi.NodeGetCapabilitiesResponse{
		Capabilities: caps,
	}, nil
}

func main() {
	ctx := context.TODO()

	proto := "unix"
	//addr := "/var/lib/kubelet/plugins/example.csi.clew.cz/csi.sock"
	addr := "/tmp/csi.sock"

	if proto == "unix" {
		if err := os.Remove(addr); err != nil && !os.IsNotExist(err) {
			log.Fatalf("failed to remove unix domain socket %s", addr)
		}
	}

	listener, err := net.Listen(proto, addr)
	if err != nil {
		log.Fatal(err)
	}

	defer listener.Close()

	server := grpc.NewServer()

	ids := newIdentityServer("example.csi.clew.cz", "1.0")
	if ids != nil {
		csi.RegisterIdentityServer(server, ids)
	}

	csi.RegisterNodeServer(server, &nodeServer{})

	go func() {
		err = server.Serve(listener)
		if err != nil {
			log.Fatal(err)
		}
	}()

	<-ctx.Done()
	server.GracefulStop()
}
