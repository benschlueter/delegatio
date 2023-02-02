/* SPDX-License-Identifier: AGPL-3.0-only
 * Copyright (c) Edgeless Systems GmbH
 * Copyright (c) Benedict Schlueter
 */

package definitions

import (
	"libvirt.org/go/libvirtxml"
)

var (
	// LibvirtStoragePoolPath is the path where the storage pool is located.
	LibvirtStoragePoolPath = "/var/lib/libvirt/images/"
	// BaseDiskName is the name for the immutable base disk, which the VMs use as COW backend.
	BaseDiskName = "delegatio"
	// DomainPrefixMaster is the prefix for the controlPlanes.
	DomainPrefixMaster = "delegatio-master-"
	// DomainPrefixWorker is the prefix for the workers.
	DomainPrefixWorker = "delegatio-worker-"
	// BootDiskName is the name of the boot drive for the VMs, the number of each VM is appended.
	BootDiskName = "delegatio-boot"
	// DiskPoolName is the name of the storage pool in which the drives are organized.
	DiskPoolName = "delegatio-pool"
	// NetworkName is the name of the network in which the VMs are connected.
	NetworkName = "delegatio-net"

	// NetworkXMLConfig is the libvirt network configuration.
	NetworkXMLConfig = libvirtxml.Network{
		Name: NetworkName,
		Forward: &libvirtxml.NetworkForward{
			Mode: "nat",
			NAT: &libvirtxml.NetworkForwardNAT{
				Ports: []libvirtxml.NetworkForwardNATPort{
					{
						Start: 1024,
						End:   65535,
					},
				},
			},
		},
		Bridge: &libvirtxml.NetworkBridge{
			Name:  "virbr1",
			STP:   "on",
			Delay: "0",
		},
		DNS: &libvirtxml.NetworkDNS{
			Enable: "yes",
		},
		IPs: []libvirtxml.NetworkIP{
			{
				Family:  "ipv4",
				Address: "10.42.0.1",
				Prefix:  16,
				DHCP: &libvirtxml.NetworkDHCP{
					Ranges: []libvirtxml.NetworkDHCPRange{
						{
							Start: "10.42.0.2",
							End:   "10.42.255.254",
						},
					},
				},
			},
		},
	}
	// PoolXMLConfig is the libvirt storage pool configuration.
	PoolXMLConfig = libvirtxml.StoragePool{
		Name:   DiskPoolName,
		Type:   "dir",
		Source: &libvirtxml.StoragePoolSource{},
		Target: &libvirtxml.StoragePoolTarget{
			Path: LibvirtStoragePoolPath,
			Permissions: &libvirtxml.StoragePoolTargetPermissions{
				Owner: "0",
				Group: "0",
				Mode:  "0777",
			},
		},
	}
	// VolumeBootXMLConfig is the libvirt storage volume configuration for the boot drive.
	VolumeBootXMLConfig = libvirtxml.StorageVolume{
		Name: BootDiskName,
		Type: "file",
		Target: &libvirtxml.StorageVolumeTarget{
			Path: LibvirtStoragePoolPath + BootDiskName,
			Format: &libvirtxml.StorageVolumeTargetFormat{
				Type: "qcow2",
			},
		},
		BackingStore: &libvirtxml.StorageVolumeBackingStore{
			Path: LibvirtStoragePoolPath + BaseDiskName,
			Format: &libvirtxml.StorageVolumeTargetFormat{
				// must be overwritten based on the used image (i.e. raw or qcow2)
				Type: "qcow2",
			},
		},
		Capacity: &libvirtxml.StorageVolumeSize{
			Unit:  "GiB",
			Value: uint64(100),
		},
	}
	// VolumeBaseXMLConfig is the libvirt storage volume configuration for the immutable base disk.
	VolumeBaseXMLConfig = libvirtxml.StorageVolume{
		Type: "file",
		Name: BaseDiskName,
		Target: &libvirtxml.StorageVolumeTarget{
			Path: LibvirtStoragePoolPath + BaseDiskName,
			Format: &libvirtxml.StorageVolumeTargetFormat{
				Type: "qcow2",
			},
		},
		Capacity: &libvirtxml.StorageVolumeSize{
			Unit:  "GiB",
			Value: uint64(100),
		},
	}

	port = uint(0)
	// DomainXMLConfig is the libvirt domain configuration.
	DomainXMLConfig = libvirtxml.Domain{
		Name: "MUST-BE-FILLED-WITH-STH-VM",
		Type: "kvm",
		Memory: &libvirtxml.DomainMemory{
			Value: 8,
			Unit:  "GiB",
		},
		Resource: &libvirtxml.DomainResource{
			Partition: "/machine",
		},
		VCPU: &libvirtxml.DomainVCPU{
			Placement: "static",
			Value:     16,
		},
		CPU: &libvirtxml.DomainCPU{
			Mode: "host-passthrough",
			Topology: &libvirtxml.DomainCPUTopology{
				Sockets: 1,
				Threads: 1,
				Cores:   16,
				Dies:    1,
			},
		},
		Features: &libvirtxml.DomainFeatureList{
			ACPI: &libvirtxml.DomainFeature{},
			PAE:  &libvirtxml.DomainFeature{},
			SMM: &libvirtxml.DomainFeatureSMM{
				State: "on",
			},
			APIC: &libvirtxml.DomainFeatureAPIC{},
			KVM:  &libvirtxml.DomainFeatureKVM{},
		},

		OS: &libvirtxml.DomainOS{
			// If Firmware is set, Loader and NVRam will be chosen automatically
			Firmware: "efi",
			FirmwareInfo: &libvirtxml.DomainOSFirmwareInfo{
				Features: []libvirtxml.DomainOSFirmwareFeature{
					{
						Name:    "secure-boot",
						Enabled: "no",
					},
				},
			},
			Type: &libvirtxml.DomainOSType{
				Arch:    "x86_64",
				Machine: "q35",
				Type:    "hvm",
			},
		},
		Devices: &libvirtxml.DomainDeviceList{
			Emulator: "/usr/bin/qemu-system-x86_64",
			Disks: []libvirtxml.DomainDisk{
				{
					Device: "disk",
					Driver: &libvirtxml.DomainDiskDriver{
						Name: "qemu",
						Type: "qcow2",
					},
					Target: &libvirtxml.DomainDiskTarget{
						Dev: "vda",
						Bus: "virtio",
					},
					Source: &libvirtxml.DomainDiskSource{
						Index: 1,
						Volume: &libvirtxml.DomainDiskSourceVolume{
							Pool:   DiskPoolName,
							Volume: BootDiskName,
						},
					},
				},
			},
			/* 			TPMs: []libvirtxml.DomainTPM{
				{
					Model: "tpm-tis",
					Backend: &libvirtxml.DomainTPMBackend{
						Emulator: &libvirtxml.DomainTPMBackendEmulator{
							Version: "2.0",
							ActivePCRBanks: &libvirtxml.DomainTPMBackendPCRBanks{
								SHA1:   &libvirtxml.DomainTPMBackendPCRBank{},
								SHA256: &libvirtxml.DomainTPMBackendPCRBank{},
								SHA384: &libvirtxml.DomainTPMBackendPCRBank{},
								SHA512: &libvirtxml.DomainTPMBackendPCRBank{},
							},
						},
					},
				},
			}, */
			Interfaces: []libvirtxml.DomainInterface{
				{
					Model: &libvirtxml.DomainInterfaceModel{
						Type: "virtio",
					},
					Source: &libvirtxml.DomainInterfaceSource{
						Network: &libvirtxml.DomainInterfaceSourceNetwork{
							Network: NetworkName,
							Bridge:  "virbr1",
						},
					},
					Alias: &libvirtxml.DomainAlias{
						Name: "net0",
					},
				},
			},
			Serials: []libvirtxml.DomainSerial{
				{
					Source: &libvirtxml.DomainChardevSource{
						Pty: &libvirtxml.DomainChardevSourcePty{
							Path: "/dev/pts/4",
						},
					},
					Target: &libvirtxml.DomainSerialTarget{
						Type: "isa-serial",
						Port: &port,
						Model: &libvirtxml.DomainSerialTargetModel{
							Name: "isa-serial",
						},
					},
				},
			},
			Consoles: []libvirtxml.DomainConsole{
				{
					TTY: "/dev/pts/4",
					Source: &libvirtxml.DomainChardevSource{
						Pty: &libvirtxml.DomainChardevSourcePty{
							Path: "/dev/pts/4",
						},
					},
					Target: &libvirtxml.DomainConsoleTarget{
						Type: "serial",
						Port: &port,
					},
				},
			},
			RNGs: []libvirtxml.DomainRNG{
				{
					Model: "virtio",
					Backend: &libvirtxml.DomainRNGBackend{
						Random: &libvirtxml.DomainRNGBackendRandom{
							Device: "/dev/urandom",
						},
					},
					Alias: &libvirtxml.DomainAlias{
						Name: "rng0",
					},
				},
			},
		},
		OnPoweroff: "destroy",
		OnCrash:    "destroy",
		OnReboot:   "restart",
	}
)
