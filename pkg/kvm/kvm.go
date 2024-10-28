package kvm

import (
	"encoding/xml"
	"fmt"
	"github.com/digitalocean/go-libvirt"
	"github.com/gpu-ninja/qcow2"
	"net/url"
	"strings"
)

type Domain struct {
	Devices Devices `xml:"devices"`
}

type Devices struct {
	Disks []Disk `xml:"disk"`
}

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

func (k *Kvm) getDomain(domainName string) (Domain, error) {
	// Find the domain by name
	domain, err := k.l.DomainLookupByName(domainName)
	if err != nil {
		return Domain{}, fmt.Errorf("error looking up the domain by name: %s", err)
	}

	// Get the domain XML
	domXML, err := k.l.DomainGetXMLDesc(domain, 0)
	if err != nil {
		return Domain{}, fmt.Errorf("error getting the domain XML: %s", err)
	}
	// Parse the XML
	var dom Domain
	err = xml.Unmarshal([]byte(domXML), &dom)
	if err != nil {
		return Domain{}, fmt.Errorf("error unmarshaling the domain XML: %s", err)
	}

	return dom, nil
}

func (k *Kvm) GetDeviceNameBySource(domainName string, sourceFile string) (string, error) {
	dom, err := k.getDomain(domainName)
	if err != nil {
		return "", err
	}

	for _, disk := range dom.Devices.Disks {
		if disk.Source.File == sourceFile {
			return disk.Target.Dev, nil
		}
	}

	return "", fmt.Errorf("can't find device for the image %s", sourceFile)
}

func (k *Kvm) getNextAvailableDevice(dom Domain) string {
	// Track used device names like sda, sdb, etc.
	usedDevices := make(map[string]bool)
	for _, disk := range dom.Devices.Disks {
		if strings.HasPrefix(disk.Target.Dev, "sd") {
			usedDevices[disk.Target.Dev] = true
		}
	}

	// Generate the next available device name (e.g., sda, sdb, sdc, ...)
	for letter := 'a'; letter <= 'z'; letter++ {
		devName := fmt.Sprintf("sd%c", letter)
		if !usedDevices[devName] {
			return devName
		}
	}
	return ""
}

func (k *Kvm) FindNextUsableDeviceName(domainName string) (string, error) {
	dom, err := k.getDomain(domainName)
	if err != nil {
		return "", err
	}

	nextDevice := k.getNextAvailableDevice(dom)
	return nextDevice, nil
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
