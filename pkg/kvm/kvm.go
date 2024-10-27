package kvm

import (
	"encoding/xml"
	"fmt"
	"github.com/digitalocean/go-libvirt"
	"github.com/gpu-ninja/qcow2"
	"net/url"
)

// Disk structure to represent a disk in the domain's XML
type Disk struct {
	XMLName xml.Name `xml:"disk"`
	Type    string   `xml:"type,attr"`
	Device  string   `xml:"device,attr"`
	Driver  struct {
		Name string `xml:"name,attr"`
		Type string `xml:"type,attr"`
	} `xml:"driver"`
	Source struct {
		File string `xml:"file,attr"`
	} `xml:"source"`
	Target struct {
		Dev string `xml:"dev,attr"`
		Bus string `xml:"bus,attr"`
	} `xml:"target"`
}

type Kvm struct {
	Uri string
	l   *libvirt.Libvirt
}

func (k *Kvm) Connect() error {
	uri, _ := url.Parse(k.Uri)
	l, err := libvirt.ConnectToURI(uri)
	if err != nil {
		return err
	}

	k.l = l
	return nil
}

func (k *Kvm) Disconnect() error {
	return k.l.Disconnect()
}

func (k *Kvm) getDomainByName(domainName string) (libvirt.Domain, error) {
	// Find the domain by name
	dom, err := k.l.DomainLookupByName(domainName)
	if err != nil {
		return libvirt.Domain{}, fmt.Errorf("error looking up the domain by name: %s", err)
	}
	// Get the domain XML
	domXML, err := k.l.DomainGetXMLDesc(dom, 0)
	if err != nil {
		return libvirt.Domain{}, fmt.Errorf("error getting the domain XML: %s", err)
	}
	// Parse the XML
	var domain libvirt.Domain
	if err := xml.Unmarshal([]byte(domXML), &domain); err != nil {
		return libvirt.Domain{}, fmt.Errorf("error unmarshalling the domain XML: %s", err)
	}
	return dom, nil
}

func (k *Kvm) prepareNewDiskXML(filepath string, targetDevice string) ([]byte, error) {
	// Create the new disk element
	newDisk := Disk{
		Type:   "file",
		Device: "disk",
	}
	newDisk.Driver.Name = "qemu"
	newDisk.Driver.Type = "qcow2"
	newDisk.Source.File = filepath    // path to your QCOW2 file
	newDisk.Target.Dev = targetDevice // The device name to be used (sdX for SCSI devices)
	newDisk.Target.Bus = "scsi"       // Use the 'scsi' bus as 'virtio' does not fully support hotplug

	// Convert the new disk back to XML
	newDiskXML, err := xml.Marshal(newDisk)
	if err != nil {
		return nil, err
	}
	return newDiskXML, nil
}

func (k *Kvm) CreateVolume(filepath string, size int64) (*qcow2.Image, error) {
	return qcow2.Create(filepath, size)
}

func (k *Kvm) AttachVolumeToDomain(domainName string, filepath string, targetDevice string) error {
	dom, err := k.getDomainByName(domainName)
	if err != nil {
		return err
	}
	newDiskXML, err := k.prepareNewDiskXML(filepath, targetDevice)
	if err != nil {
		return fmt.Errorf("error preparing the new disk XML: %s", err)
	}
	err = k.l.DomainAttachDevice(dom, string(newDiskXML))
	if err != nil {
		return fmt.Errorf("error attaching the device: %s", err)
	}
	return nil
}

func (k *Kvm) DetachVolumeFromDomain(domainName string, filepath string, targetDevice string) error {
	dom, err := k.getDomainByName(domainName)
	if err != nil {
		return err
	}
	newDiskXML, err := k.prepareNewDiskXML(filepath, targetDevice)
	if err != nil {
		return err
	}
	err = k.l.DomainDetachDevice(dom, string(newDiskXML))
	if err != nil {
		return err
	}
	return nil
}
