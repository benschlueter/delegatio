package main

import (
	"flag"
	"fmt"
	"io"
	"os"
	"os/signal"
	"strings"
	"syscall"
	"time"

	"go.uber.org/multierr"
	"go.uber.org/zap"
	"libvirt.org/go/libvirt"
)

// Usage:
// go build
//./image-measurement --path=disk.raw --type=raw

type libvirtInstance struct {
	conn      *libvirt.Connect
	log       *zap.Logger
	imagePath string
}

func (l *libvirtInstance) uploadBaseImage(baseVolume *libvirt.StorageVol) (err error) {
	stream, err := l.conn.NewStream(libvirt.STREAM_NONBLOCK)
	if err != nil {
		return err
	}
	defer func() { _ = stream.Free() }()
	file, err := os.Open(l.imagePath)
	if err != nil {
		return fmt.Errorf("error while opening %s: %s", l.imagePath, err)
	}
	defer func() {
		err = multierr.Append(err, file.Close())
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
	for {
		_, err := file.Read(buffer)
		if err != nil && err != io.EOF {
			return err
		}
		if err == io.EOF {
			break
		}
		num, err := stream.Send(buffer)
		if err != nil {
			return err
		}
		transferredBytes += num

	}
	if transferredBytes < int(fi.Size()) {
		return fmt.Errorf("only send %d out of %d bytes", transferredBytes, fi.Size())
	}
	return nil
}

func (l *libvirtInstance) createStoragePool() (*libvirt.StoragePool, error) {
	poolXMLString, err := poolXMLConfig.Marshal()
	if err != nil {
		return nil, err
	}

	l.log.Info("creating storage pool")
	poolObject, err := l.conn.StoragePoolDefineXML(poolXMLString, libvirt.STORAGE_POOL_DEFINE_VALIDATE)
	if err != nil {
		return nil, fmt.Errorf("error defining libvirt storage pool: %s", err)
	}
	if err := poolObject.Build(libvirt.STORAGE_POOL_BUILD_NEW); err != nil {
		return nil, fmt.Errorf("error building libvirt storage pool: %s", err)
	}
	if err := poolObject.Create(libvirt.STORAGE_POOL_CREATE_NORMAL); err != nil {
		return nil, fmt.Errorf("error creating libvirt storage pool: %s", err)
	}
	return poolObject, nil
}

func (l *libvirtInstance) createBaseImage(storagePool *libvirt.StoragePool) error {
	volumeBaseXMLString, err := volumeBaseXMLConfig.Marshal()
	if err != nil {
		return err
	}
	volumeBaseObject, err := storagePool.StorageVolCreateXML(volumeBaseXMLString, 0)
	if err != nil {
		return fmt.Errorf("error creating libvirt storage volume 'base': %s", err)
	}
	defer func() { _ = volumeBaseObject.Free() }()

	l.log.Info("uploading image to libvirt")
	return l.uploadBaseImage(volumeBaseObject)
}

func (l *libvirtInstance) createBootImage(storagePool *libvirt.StoragePool) error {
	volumeBootXMLString, err := volumeBootXMLConfig.Marshal()
	if err != nil {
		return err
	}

	l.log.Info("creating storage volume 'boot'")
	bootVol, err := storagePool.StorageVolCreateXML(volumeBootXMLString, 0)
	if err != nil {
		return fmt.Errorf("error creating libvirt storage volume 'boot': %s", err)
	}
	defer func() { _ = bootVol.Free() }()

	return nil
}

func (l *libvirtInstance) createNetwork() error {
	networkXMLString, err := networkXMLConfig.Marshal()
	if err != nil {
		return err
	}
	l.log.Info("creating network")
	network, err := l.conn.NetworkCreateXML(networkXMLString)
	if err != nil {
		return err
	}
	defer func() { _ = network.Free() }()
	return nil
}

func (l *libvirtInstance) createDomain() error {
	domainXMLString, err := domainXMLConfig.Marshal()
	if err != nil {
		return err
	}
	l.log.Info("creating domain")
	domain, err := l.conn.DomainCreateXML(domainXMLString, libvirt.DOMAIN_NONE)
	if err != nil {
		return fmt.Errorf("error creating libvirt domain: %s", err)
	}
	defer func() { _ = domain.Free() }()
	return nil
}

func (l *libvirtInstance) deleteNetwork() error {
	nets, err := l.conn.ListAllNetworks(libvirt.CONNECT_LIST_NETWORKS_ACTIVE)
	if err != nil {
		return err
	}
	defer func() {
		for _, net := range nets {
			_ = net.Free()
		}
	}()
	for _, net := range nets {
		name, err := net.GetName()
		if err != nil {
			return err
		}
		if name != networkName {
			continue
		}
		if err := net.Destroy(); err != nil {
			return err
		}
	}
	return nil
}

func (l *libvirtInstance) deleteDomain() error {
	doms, err := l.conn.ListAllDomains(libvirt.CONNECT_LIST_DOMAINS_ACTIVE)
	if err != nil {
		return err
	}
	defer func() {
		for _, dom := range doms {
			_ = dom.Free()
		}
	}()
	for _, dom := range doms {
		name, err := dom.GetName()
		if err != nil {
			return err
		}
		if name != domainName {
			continue
		}
		if err := dom.Destroy(); err != nil {
			return err
		}
	}
	return nil
}

func (l *libvirtInstance) deleteVolume(pool *libvirt.StoragePool) error {
	volumes, err := pool.ListAllStorageVolumes(0)
	if err != nil {
		return err
	}
	defer func() {
		for _, volume := range volumes {
			_ = volume.Free()
		}
	}()
	for _, volume := range volumes {
		name, err := volume.GetName()
		if err != nil {
			return err
		}
		if !strings.Contains(name, bootDiskName) && name != baseDiskName {
			continue
		}
		if err := volume.Delete(libvirt.STORAGE_VOL_DELETE_NORMAL); err != nil {
			return err
		}
	}
	return nil
}

func (l *libvirtInstance) deletePool() error {
	pools, err := l.conn.ListAllStoragePools(libvirt.CONNECT_LIST_STORAGE_POOLS_DIR)
	if err != nil {
		return err
	}
	defer func() {
		for _, pool := range pools {
			_ = pool.Free()
		}
	}()
	for _, pool := range pools {
		name, err := pool.GetName()
		if err != nil {
			return err
		}
		if name != diskPoolName {
			continue
		}
		active, err := pool.IsActive()
		if err != nil {
			return err
		}
		if active {
			if err := l.deleteVolume(&pool); err != nil {
				return err
			}
			if err := pool.Destroy(); err != nil {
				return err
			}
			if err := pool.Delete(libvirt.STORAGE_POOL_DELETE_NORMAL); err != nil {
				return err
			}
		}
		// If something fails and the Pool becomes inactive, we cannot delete/destroy it anymore.
		// We have to undefine it in this case
		if err := pool.Undefine(); err != nil {
			return err
		}
	}
	return nil
}

func (l *libvirtInstance) deleteLibvirtInstance() error {
	var err error
	err = multierr.Append(err, l.deleteNetwork())
	err = multierr.Append(err, l.deleteDomain())
	err = multierr.Append(err, l.deletePool())
	return err
}

func (l *libvirtInstance) waitForQemuConnection(sig <-chan os.Signal) {
	cb := func(c *libvirt.Connect, d *libvirt.Domain, event *libvirt.DomainEventAgentLifecycle) {
		name, err := d.GetName()
		if err != nil {
			l.log.Error("error in callback function cannot obtain name", zap.Error(err))
			return
		}
		l.log.Info("qemu guest agent becomes ready", zap.String("name", name))
	}
	fd, err := l.conn.DomainEventAgentLifecycleRegister(nil, cb)
	if err != nil {
		l.log.DPanic("error getting domains", zap.Error(err))
	}
	l.log.Info("registered Callback", zap.Int("fd", fd))
}

func (l *libvirtInstance) executeCommands() (err error) {
	// sanity check
	if err := l.deleteLibvirtInstance(); err != nil {
		return err
	}
	done := make(chan struct{}, 1)
	defer func() {
		err = multierr.Append(err, l.deleteLibvirtInstance())
	}()
	pool, err := l.createStoragePool()
	if err != nil {
		return err
	}
	defer func() { _ = pool.Free() }()
	if err := l.createBaseImage(pool); err != nil {
		return err
	}
	if err := l.createBootImage(pool); err != nil {
		return err
	}
	if err := l.createNetwork(); err != nil {
		return err
	}
	if err := l.createDomain(); err != nil {
		return err
	}

	sigs := make(chan os.Signal, 1)
	signal.Notify(sigs, syscall.SIGINT, syscall.SIGTERM)

	l.waitForQemuConnection(sigs)

	time.Sleep(10 * time.Second)

	domain, err := l.conn.LookupDomainByName("delegatio-vm")
	if err != nil {
		l.log.With(zap.Error(err)).DPanic("Failed to connect to guest")
	}
	defer domain.Free()
	name, err := domain.GetGuestInfo(libvirt.DOMAIN_GUEST_INFO_HOSTNAME, 0)
	if err != nil {
		l.log.With(zap.Error(err)).DPanic("Failed to connect to guest")
	}
	l.log.Info(name.Hostname)
	select {
	case <-sigs:
		break
	}

	signal.Stop(sigs)
	close(sigs)
	close(done)
	return nil
}

func main() {
	var imageLocation string
	var imageType string
	flag.StringVar(&imageLocation, "path", "", "path to the image to measure (required)")
	flag.StringVar(&imageType, "type", "", "type of the image. One of 'qcow2' or 'raw' (required)")
	flag.Parse()
	zapconf := zap.NewDevelopmentConfig()
	log, err := zapconf.Build()
	if err != nil {
		log.With(zap.Error(err)).DPanic("Failed to create logger")
	}

	if imageLocation == "" || imageType == "" {
		flag.Usage()
		os.Exit(1)
	}
	volumeBootXMLConfig.BackingStore.Format.Type = imageType

	if err := libvirt.EventRegisterDefaultImpl(); err != nil {
		log.With(zap.Error(err)).DPanic("Failed to create event listener")
	}
	go func() {
		for {
			fmt.Println("run go routine")
			err := libvirt.EventRunDefaultImpl()
			if err != nil {
				log.With(zap.Error(err)).DPanic("go func failed")
			}
		}
	}()
	conn, err := libvirt.NewConnect("qemu:///system")
	if err != nil {
		log.With(zap.Error(err)).DPanic("Failed to connect to libvirt")
	}
	defer conn.Close()

	lInstance := libvirtInstance{
		conn:      conn,
		log:       log,
		imagePath: imageLocation,
	}

	if err := lInstance.executeCommands(); err != nil {
		log.With(zap.Error(err)).DPanic("Failed to obtain PCR measurements")
	}
	log.Info("instaces terminated successfully")
}
