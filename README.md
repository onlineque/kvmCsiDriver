
# KVM CSI Driver

Implementation of a CSI driver for KVM. Originally it was aimed just to understand the CSI concept and how it works. It has not been well tested and is highly likely not full-featured. Anyway, this PoC works ! :-)
## Features and Architecture

These are the result of studying of the [Kubernetes CSI Developer Documentation](https://kubernetes-csi.github.io/docs/introduction.html). There are three components making up the KVM CSI Driver:

- Storage Agent - this component runs on KVM host and is responsible for QCOW2 image creation / deleting and attaching / detaching to/from KVM domains (virtual machines).
- Controller Server - runs as a Deployment with 2 replicas inside the Kubernetes cluster. It has a sidecar container running the [csi-provisioner](https://github.com/kubernetes-csi/external-provisioner) and watches for new Persistent Volume Claims (PVCs). It calls the CSI Driver  ( by calling `CreateVolume`). That calls the Storage Agent and a new QCOW2 image is then created. It also calls the CSI Driver (by calling `DeleteVolume`) in case the volume is not needed anymore. That calls again the Storage Agent and triggers the deleting of the QCOW2 image.
- Node Server - runs as a DaemonSet on every worker inside the Kubernetes cluster. `NodePublishVolume` is called when the volume (already created on KVM through Controller Server) is requested to be published - attached to the desired node, formatted and mounted inside the node at the requested mountpoint. `NodeUnpublishVolume` is called when the mounted volume is not needed anymore and can be unmounted from the node and detached from it.


## Prerequisites

All worker nodes on the Kubernetes cluster have to be labeled with:
```bash
  kubectl label node <node_name> example.clew.cz/kvm-domain=<kvm_domain_name_hosting_this_k8s_node>
```
The value needs to contain the name of the KVM domain (virtual machine) which runs this Kubernetes node.

## Deployment

To deploy the KVM CSI Driver you need to install the Storage Agent component on the KVM host where the Kubernetes cluster is running. Only a single host KVM is supported with the Persistent Volumes created as QCOW2 images.
Download the latest storageagent component from https://github.com/onlineque/kvmCsiDriver/releases, save it to /usr/local/bin/storageagent.
Set the executable rights:

```bash
  chmod +x /usr/local/bin/storageagent
```

Copy the SystemD unit file (from storageagent/storageagent.service) to /etc/systemd/system/
Enable the service to run upon KVM host start and start it:
```bash
  systemctl enable --now storageagent.service
```
Deploy the KVM CSI Driver inside your Kubernetes cluster with helm, replace the <storage_agent_FQDN> with the actual FQDN (fully qualified domain name) of the KVM machine where your storageagent is running:
```bash
  helm install --create-namespace -n kvm-csi-driver kvm-csi-driver oci://ghcr.io/onlineque/kvm-csi-driver --set storageAgent.target=<storage_agent_FQDN>:7003
```

## Roadmap

- `StageVolume` and `UnstageVolume` so the attaching and formatting of the disk is done before publishing it
- testing
- Sanity testing, probably with [CSI Sanity](https://github.com/kubernetes-csi/csi-test/tree/master/cmd/csi-sanity)
- 🐛 bug hunting