package driver

import (
	"context"
	"fmt"
	"github.com/akutz/gofsutil"
	csi "github.com/onlineque/kvmCsiDriver/csi_proto"
	sa "github.com/onlineque/kvmCsiDriver/storageagent_proto"
	"google.golang.org/grpc"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/credentials/insecure"
	"google.golang.org/grpc/status"
	wrappers "google.golang.org/protobuf/types/known/wrapperspb"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/client-go/kubernetes"
	"k8s.io/client-go/rest"
	"log"
	"net"
	"os"
	"time"
)

const ImplementMe = "implement me"

type controllerServer struct {
	csi.UnimplementedControllerServer
}

type nodeServer struct {
	nodeID string
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

func (ids *identityServer) GetPluginInfo(_ context.Context, _ *csi.GetPluginInfoRequest) (*csi.GetPluginInfoResponse, error) {
	if ids.name == "" {
		return nil, status.Error(codes.Unavailable, "driver name not configured")
	}
	if ids.version == "" {
		return nil, status.Error(codes.Unavailable, "driver version not configured")
	}
	log.Print("GetPluginInfo called")
	return &csi.GetPluginInfoResponse{
		Name:          ids.name,
		VendorVersion: ids.version,
	}, nil
}

func (ids *identityServer) GetPluginCapabilities(_ context.Context, _ *csi.GetPluginCapabilitiesRequest) (*csi.GetPluginCapabilitiesResponse, error) {
	log.Print("GetPluginCapabilities called")
	return &csi.GetPluginCapabilitiesResponse{
		Capabilities: []*csi.PluginCapability{
			{
				Type: &csi.PluginCapability_Service_{
					Service: &csi.PluginCapability_Service{
						Type: csi.PluginCapability_Service_CONTROLLER_SERVICE,
					},
				},
			},
		},
	}, nil
}

func (ids *identityServer) Probe(_ context.Context, _ *csi.ProbeRequest) (*csi.ProbeResponse, error) {
	log.Print("Probe called")
	return &csi.ProbeResponse{Ready: &wrappers.BoolValue{Value: true}}, nil
}

func (ns *nodeServer) NodePublishVolume(ctx context.Context, req *csi.NodePublishVolumeRequest) (*csi.NodePublishVolumeResponse, error) {
	// mounting the volume should be here
	log.Print("NodePublishVolume called")
	volumeID := req.VolumeId
	targetPath := req.TargetPath
	log.Printf("- volumeId: %s", volumeID)
	log.Printf("  targetPath: %s", targetPath)

	// attach volume to this node
	config, err := rest.InClusterConfig()
	if err != nil {
		return nil, err
	}
	clientset, err := kubernetes.NewForConfig(config)
	if err != nil {
		return nil, err
	}

	nodeObj, err := clientset.CoreV1().Nodes().Get(context.TODO(), os.Getenv("NODE_ID"), metav1.GetOptions{})
	if err != nil {
		return nil, err
	}
	kvmDomain := nodeObj.Labels["example.clew.cz/kvm-domain"]
	log.Printf("  kvmNode: %s", kvmDomain)

	conn, err := grpc.NewClient(os.Getenv("STORAGEAGENT_TARGET"), grpc.WithTransportCredentials(insecure.NewCredentials()))
	if err != nil {
		return nil, err
	}
	defer conn.Close()

	c := sa.NewStorageAgentClient(conn)

	// ctx, cancel := context.WithTimeout(context.Background(), time.Second)
	// defer cancel()

	img, err := c.AttachVolume(ctx, &sa.VolumeRequest{
		ImageId:    volumeID,
		TargetPath: targetPath,
		DomainName: kvmDomain,
	})
	if err != nil {
		return nil, err
	}
	log.Printf("successfully attached volume %s to %s:%s", img.ImageId, kvmDomain, targetPath)

	// create filesystem (first check if it's not there already ?)
	// mount it into targetPath
	log.Printf("checking filesystem on /dev/%s", img.Device)
	// create mount directory
	if _, err := os.Stat(targetPath); os.IsNotExist(err) {
		// Step 2: Create the directory along with any necessary parents
		err := os.MkdirAll(targetPath, 0755) // 0755 gives read, write, and execute permissions to the owner, and read + execute permissions to others
		if err != nil {
			return nil, fmt.Errorf("failed to create the mountpoint directory: %w", err)
		}
		log.Printf("created mount point directory: %s\n", targetPath)
	}

	err = gofsutil.FormatAndMount(ctx, fmt.Sprintf("/dev/%s", img.Device), targetPath, "ext4")
	if err != nil {
		return nil, err
	}

	return &csi.NodePublishVolumeResponse{}, nil
}

func (ns *nodeServer) NodeUnpublishVolume(ctx context.Context, req *csi.NodeUnpublishVolumeRequest) (*csi.NodeUnpublishVolumeResponse, error) {
	log.Print("NodeUnpublishVolume called")

	volumeId := req.VolumeId
	targetPath := req.TargetPath
	log.Printf("- volumeId: %s", volumeId)
	log.Printf("  targetPath: %s", targetPath)

	// unmounting  the volume should be here
	err := gofsutil.Unmount(ctx, req.TargetPath)
	if err != nil {
		return nil, err
	}

	// detach volume from this node
	config, err := rest.InClusterConfig()
	if err != nil {
		return nil, err
	}
	clientset, err := kubernetes.NewForConfig(config)
	if err != nil {
		return nil, err
	}

	nodeObj, err := clientset.CoreV1().Nodes().Get(context.TODO(), os.Getenv("NODE_ID"), metav1.GetOptions{})
	if err != nil {
		return nil, err
	}
	kvmDomain := nodeObj.Labels["example.clew.cz/kvm-domain"]
	log.Printf("  kvmNode: %s", kvmDomain)

	conn, err := grpc.NewClient(os.Getenv("STORAGEAGENT_TARGET"), grpc.WithTransportCredentials(insecure.NewCredentials()))
	if err != nil {
		return nil, err
	}
	defer conn.Close()

	c := sa.NewStorageAgentClient(conn)

	ctx, cancel := context.WithTimeout(context.Background(), time.Second)
	defer cancel()

	img, err := c.DetachVolume(ctx, &sa.VolumeRequest{
		ImageId:    volumeId,
		TargetPath: targetPath,
		DomainName: kvmDomain,
	})
	if err != nil {
		return nil, err
	}
	log.Printf("successfully detached volume %s to %s:%s", img.ImageId, kvmDomain, targetPath)

	return &csi.NodeUnpublishVolumeResponse{}, nil
}

func (ns *nodeServer) NodeGetCapabilities(_ context.Context, _ *csi.NodeGetCapabilitiesRequest) (*csi.NodeGetCapabilitiesResponse, error) {
	log.Print("NodeGetCapabilities called")
	caps := []*csi.NodeServiceCapability{}

	return &csi.NodeGetCapabilitiesResponse{
		Capabilities: caps,
	}, nil
}

func (ns *nodeServer) NodeGetInfo(_ context.Context, _ *csi.NodeGetInfoRequest) (*csi.NodeGetInfoResponse, error) {
	log.Print("NodeGetInfo called")
	return &csi.NodeGetInfoResponse{
		NodeId: ns.nodeID,
	}, nil

}

func (cs *controllerServer) CreateVolume(ctx context.Context, req *csi.CreateVolumeRequest) (*csi.CreateVolumeResponse, error) {
	log.Print("CreateVolume called")
	topologies := []*csi.Topology{}

	log.Printf("- name: %s", req.Name)
	log.Printf("  required capacity: %d", req.CapacityRange.RequiredBytes)
	log.Printf("  parameters: %v", req.GetParameters())

	volumeId := req.GetParameters()["csi.storage.k8s.io/pv/name"]

	conn, err := grpc.NewClient(os.Getenv("STORAGEAGENT_TARGET"), grpc.WithTransportCredentials(insecure.NewCredentials()))
	if err != nil {
		return nil, err
	}
	defer conn.Close()

	c := sa.NewStorageAgentClient(conn)

	// ctx, cancel := context.WithTimeout(context.Background(), time.Second)
	// defer cancel()

	img, err := c.CreateImage(ctx, &sa.ImageRequest{
		ImageId: volumeId,
		Size:    req.CapacityRange.RequiredBytes,
	})
	if err != nil {
		return nil, err
	}

	return &csi.CreateVolumeResponse{
		Volume: &csi.Volume{
			VolumeId:           img.ImageId,
			CapacityBytes:      req.CapacityRange.RequiredBytes,
			VolumeContext:      req.GetParameters(),
			ContentSource:      req.GetVolumeContentSource(),
			AccessibleTopology: topologies,
		},
	}, nil
}

func (cs *controllerServer) DeleteVolume(ctx context.Context, req *csi.DeleteVolumeRequest) (*csi.DeleteVolumeResponse, error) {
	log.Print("DeleteVolume called")
	volumeId := req.VolumeId
	log.Printf("- name: %s", volumeId)

	conn, err := grpc.NewClient(os.Getenv("STORAGEAGENT_TARGET"), grpc.WithTransportCredentials(insecure.NewCredentials()))
	if err != nil {
		return nil, err
	}
	defer conn.Close()

	c := sa.NewStorageAgentClient(conn)

	// ctx, cancel := context.WithTimeout(context.Background(), time.Second)
	// defer cancel()

	_, err = c.DeleteImage(ctx, &sa.ImageRequest{
		ImageId: volumeId,
	})
	if err != nil {
		return nil, err
	}

	return &csi.DeleteVolumeResponse{}, nil
}

func (cs *controllerServer) ControllerPublishVolume(_ context.Context, _ *csi.ControllerPublishVolumeRequest) (*csi.ControllerPublishVolumeResponse, error) {
	log.Print("ControllerPublishVolume called")
	return &csi.ControllerPublishVolumeResponse{
		PublishContext: map[string]string{},
	}, nil
}

func (cs *controllerServer) ControllerUnpublishVolume(_ context.Context, _ *csi.ControllerUnpublishVolumeRequest) (*csi.ControllerUnpublishVolumeResponse, error) {
	log.Print("ControllerUnpublishVolume called")
	return &csi.ControllerUnpublishVolumeResponse{}, nil
}

func (cs *controllerServer) ValidateVolumeCapabilities(_ context.Context, _ *csi.ValidateVolumeCapabilitiesRequest) (*csi.ValidateVolumeCapabilitiesResponse, error) {
	//TODO implement me
	panic(ImplementMe)
}

func (cs *controllerServer) ListVolumes(_ context.Context, _ *csi.ListVolumesRequest) (*csi.ListVolumesResponse, error) {
	//TODO implement me
	panic(ImplementMe)
}

func (cs *controllerServer) GetCapacity(_ context.Context, _ *csi.GetCapacityRequest) (*csi.GetCapacityResponse, error) {
	//TODO implement me
	panic(ImplementMe)
}

func (cs *controllerServer) ControllerGetCapabilities(_ context.Context, _ *csi.ControllerGetCapabilitiesRequest) (*csi.ControllerGetCapabilitiesResponse, error) {
	log.Print("ControllerGetCapabilities called")
	var csc []*csi.ControllerServiceCapability
	csc = append(csc, &csi.ControllerServiceCapability{
		Type: &csi.ControllerServiceCapability_Rpc{
			Rpc: &csi.ControllerServiceCapability_RPC{
				Type: csi.ControllerServiceCapability_RPC_CREATE_DELETE_VOLUME,
			},
		},
	})

	return &csi.ControllerGetCapabilitiesResponse{
		Capabilities: csc,
	}, nil
}

func (cs *controllerServer) CreateSnapshot(_ context.Context, _ *csi.CreateSnapshotRequest) (*csi.CreateSnapshotResponse, error) {
	//TODO implement me
	panic(ImplementMe)
}

func (cs *controllerServer) DeleteSnapshot(_ context.Context, _ *csi.DeleteSnapshotRequest) (*csi.DeleteSnapshotResponse, error) {
	//TODO implement me
	panic(ImplementMe)
}

func (cs *controllerServer) ListSnapshots(_ context.Context, _ *csi.ListSnapshotsRequest) (*csi.ListSnapshotsResponse, error) {
	//TODO implement me
	panic(ImplementMe)
}

func (cs *controllerServer) ControllerExpandVolume(_ context.Context, _ *csi.ControllerExpandVolumeRequest) (*csi.ControllerExpandVolumeResponse, error) {
	//TODO implement me
	panic(ImplementMe)
}

func (cs *controllerServer) ControllerGetVolume(_ context.Context, _ *csi.ControllerGetVolumeRequest) (*csi.ControllerGetVolumeResponse, error) {
	//TODO implement me
	panic(ImplementMe)
}

func (cs *controllerServer) ControllerModifyVolume(_ context.Context, _ *csi.ControllerModifyVolumeRequest) (*csi.ControllerModifyVolumeResponse, error) {
	//TODO implement me
	panic(ImplementMe)
}

func RunServer(runControllerServer bool, runNodeServer bool) {
	ctx := context.TODO()

	proto := "unix"
	addr := "/csi/csi.sock"
	//addr := "/tmp/csi.sock"

	if err := os.Remove(addr); err != nil && !os.IsNotExist(err) {
		log.Fatalf("failed to remove unix domain socket %s", addr)
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

	if runNodeServer {
		csi.RegisterNodeServer(server, &nodeServer{
			nodeID: os.Getenv("NODE_ID"),
		})
	}

	if runControllerServer {
		csi.RegisterControllerServer(server, &controllerServer{})
	}

	go func() {
		err = server.Serve(listener)
		if err != nil {
			log.Fatal(err)
		}
	}()

	<-ctx.Done()
	server.GracefulStop()
}
