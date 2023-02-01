/* SPDX-License-Identifier: AGPL-3.0-only
* Copyright (c) Edgeless Systems GmbH
* Copyright (c) Benedict Schlueter
 */

package qemu

import (
	"context"
	"errors"
	"fmt"
	"io"
	"strconv"

	"github.com/benschlueter/delegatio/internal/config/definitions"
	"github.com/benschlueter/delegatio/internal/infrastructure/configurer"
	"go.uber.org/zap"
	"libvirt.org/go/libvirt"
)

func (l *libvirtInstance) uploadBaseImage(ctx context.Context, baseVolume storageVolume) (err error) {
	stream, err := l.Conn.NewStream(libvirt.STREAM_NONBLOCK)
	if err != nil {
		return err
	}
	defer func() { _ = stream.Free() }()
	file, err := l.fs.Open(l.ImagePath)
	if err != nil {
		return fmt.Errorf("error while opening %s: %s", l.ImagePath, err)
	}
	defer func() {
		err = errors.Join(err, file.Close())
	}()

	fi, err := file.Stat()
	if err != nil {
		return err
	}
	if err := baseVolume.Upload(stream, 0, uint64(fi.Size()), 0); err != nil {
		return err
	}
	transferredBytes := 0
	buffer := make([]byte, 4*1024*1024)

	// Fill the stream with buffer-chunks of the image.
	// Since this can take long we must make this interruptable in case of
	// a context cancellation.
loop:
	for {
		select {
		case <-ctx.Done():
			err := stream.Abort()
			l.Log.Info("context cancel during image upload", zap.Error(err))
			return ctx.Err()
		default:
			_, err := file.Read(buffer)
			if err != nil && err != io.EOF {
				return err
			}
			if err == io.EOF {
				break loop
			}
			num, err := stream.Send(buffer)
			if err != nil {
				return err
			}
			transferredBytes += num
		}
	}
	/* 	if err := stream.Finish(); err != nil {
		return err
	} */
	if transferredBytes < int(fi.Size()) {
		return fmt.Errorf("only send %d out of %d bytes", transferredBytes, fi.Size())
	}
	l.Log.Info("image upload successful", zap.Int64("image size", fi.Size()), zap.Int("transferred bytes", transferredBytes))
	return nil
}

func deleteVolumesFromPool(pool storagePool) error {
	volumes, err := pool.ListAllStorageVolumes(0)
	if err != nil {
		return err
	}
	defer func() {
		for _, volume := range volumes {
			_ = volume.Free()
		}
	}()
	// We own the pool, so we can simply delete all volumes in it.
	for _, volume := range volumes {
		if err := volume.Delete(libvirt.STORAGE_VOL_DELETE_NORMAL); err != nil {
			return fmt.Errorf("volume delete %w", err)
		}
	}
	return nil
}

func (l *libvirtInstance) createAgent(ctx context.Context) (*configurer.Configurer, error) {
	controlPlaneIP, err := l.getControlPlaneIP()
	if err != nil {
		return nil, err
	}
	workerInstance, err := l.getWorkerInstanceIPs(ctx)
	if err != nil {
		return nil, err
	}
	vmAgent, err := configurer.NewConfigurer(l.Log, controlPlaneIP, workerInstance)
	if err != nil {
		return nil, err
	}
	return vmAgent, nil
}

func (l *libvirtInstance) getControlPlaneIP() (ip string, err error) {
	domain, err := l.Conn.LookupDomainByName(definitions.DomainPrefixMaster + "0")
	if err != nil {
		return
	}
	defer func() { _ = domain.Free() }()
	iface, err := domain.ListAllInterfaceAddresses(libvirt.DOMAIN_INTERFACE_ADDRESSES_SRC_LEASE)
	if err != nil {
		return
	}
	for _, netInterface := range iface {
		if netInterface.Name == "lo" {
			continue
		}
		for _, addr := range netInterface.Addrs {
			if addr.Type == libvirt.IP_ADDR_TYPE_IPV4 {
				ip = addr.Addr
			}
		}
	}
	if len(ip) == 0 {
		return "", fmt.Errorf("could not find ip addr of domain")
	}
	return
}

func (l *libvirtInstance) getWorkerInstanceIPs(ctx context.Context) (map[string]string, error) {
	nameToIP := make(map[string]string)
	for id := 0; id < l.workerNodeNum; id++ {
		node := definitions.DomainPrefixWorker + strconv.Itoa(id)
		ip, err := l.blockUntilNetworkIsReady(ctx, node)
		if err != nil {
			return nil, err
		}
		nameToIP[node] = ip
	}
	return nameToIP, nil
}
