/* SPDX-License-Identifier: AGPL-3.0-only
 * Copyright (c) Edgeless Systems GmbH
 * Copyright (c) Benedict Schlueter
 */

package definitions

import (
	"libvirt.org/go/libvirtxml"
)

var (
	LibvirtStoragePoolPath = "/var/lib/libvirt/images/"
	BaseDiskName           = "delegatio"
	BootDiskName           = "delegatio-boot"
	DiskPoolName           = "delegatio-pool"
	NetworkName            = "delegatio-net"

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

	port            = uint(0)
	DomainXMLConfig = libvirtxml.Domain{
		Name: "MUST-BE-FILLED-WITH-STH-VM",
		Type: "kvm",
		Memory: &libvirtxml.DomainMemory{
			Value: 4,
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
			// Needed for qemu guest agent
			Channels: []libvirtxml.DomainChannel{
				{
					Source: &libvirtxml.DomainChardevSource{
						UNIX: &libvirtxml.DomainChardevSourceUNIX{
							Mode: "bind",
						},
					},
					Target: &libvirtxml.DomainChannelTarget{
						VirtIO: &libvirtxml.DomainChannelTargetVirtIO{
							Name: "org.qemu.guest_agent.0",
						},
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
