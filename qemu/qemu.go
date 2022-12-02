package qemu

import (
	"fmt"
	"io"
	"os"
	"os/signal"
	"path"
	"strconv"
	"syscall"

	"github.com/benschlueter/delegatio/test/qemu/definitions"
	"go.uber.org/multierr"
	"go.uber.org/zap"
	"libvirt.org/go/libvirt"
)

type LibvirtInstance struct {
	Conn               *libvirt.Connect
	Log                *zap.Logger
	ImagePath          string
	registeredDomains  []string
	registeredNetworks []string
	registeredPools    []string
	registeredDisks    []string
}

func (l *LibvirtInstance) uploadBaseImage(baseVolume *libvirt.StorageVol) (err error) {
	stream, err := l.Conn.NewStream(libvirt.STREAM_NONBLOCK)
	if err != nil {
		return err
	}
	defer func() { _ = stream.Free() }()
	file, err := os.Open(l.ImagePath)
	if err != nil {
		return fmt.Errorf("error while opening %s: %s", l.ImagePath, err)
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

func (l *LibvirtInstance) CreateStoragePool() error {
	// Might be needed for renaming in the future
	poolXMLCopy := definitions.PoolXMLConfig
	poolXMLString, err := poolXMLCopy.Marshal()
	if err != nil {
		return err
	}

	l.Log.Info("creating storage pool")
	poolObject, err := l.Conn.StoragePoolDefineXML(poolXMLString, libvirt.STORAGE_POOL_DEFINE_VALIDATE)
	if err != nil {
		return fmt.Errorf("error defining libvirt storage pool: %s", err)
	}
	if err := poolObject.Build(libvirt.STORAGE_POOL_BUILD_NEW); err != nil {
		return fmt.Errorf("error building libvirt storage pool: %s", err)
	}
	if err := poolObject.Create(libvirt.STORAGE_POOL_CREATE_NORMAL); err != nil {
		return fmt.Errorf("error creating libvirt storage pool: %s", err)
	}
	defer func() { _ = poolObject.Free() }()
	l.registeredPools = append(l.registeredPools, poolXMLCopy.Name)
	return nil
}

func (l *LibvirtInstance) CreateBaseImage() error {
	volumeBaseXMLString, err := definitions.VolumeBaseXMLConfig.Marshal()
	if err != nil {
		return err
	}
	storagePool, err := l.Conn.LookupStoragePoolByTargetPath(definitions.LibvirtStoragePoolPath)
	if err != nil {
		return err
	}
	defer func() { _ = storagePool.Free() }()
	l.Log.Info("creating base storage image")
	volumeBaseObject, err := storagePool.StorageVolCreateXML(volumeBaseXMLString, 0)
	if err != nil {
		return fmt.Errorf("error creating libvirt storage volume 'base': %s", err)
	}
	defer func() { _ = volumeBaseObject.Free() }()
	l.registeredDisks = append(l.registeredDisks, definitions.VolumeBaseXMLConfig.Name)

	l.Log.Info("uploading image to libvirt")
	return l.uploadBaseImage(volumeBaseObject)
}

func (l *LibvirtInstance) CreateBootImage(id string) error {
	volumeBootXMLCopy := definitions.VolumeBootXMLConfig
	volumeBootXMLCopy.Name = id
	volumeBootXMLCopy.Target.Path = path.Join(definitions.LibvirtStoragePoolPath, id)

	volumeBootXMLString, err := volumeBootXMLCopy.Marshal()
	if err != nil {
		return err
	}
	storagePool, err := l.Conn.LookupStoragePoolByTargetPath(definitions.LibvirtStoragePoolPath)
	if err != nil {
		return err
	}
	defer func() { _ = storagePool.Free() }()
	l.Log.Info("creating storage volume 'boot'")
	bootVol, err := storagePool.StorageVolCreateXML(volumeBootXMLString, 0)
	if err != nil {
		return fmt.Errorf("error creating libvirt storage volume 'boot': %s", err)
	}
	defer func() { _ = bootVol.Free() }()
	l.registeredDisks = append(l.registeredDisks, volumeBootXMLCopy.Name)
	return nil
}

func (l *LibvirtInstance) CreateNetwork() error {
	networkXMLString, err := definitions.NetworkXMLConfig.Marshal()
	if err != nil {
		return err
	}
	l.Log.Info("creating network")
	network, err := l.Conn.NetworkCreateXML(networkXMLString)
	if err != nil {
		return err
	}
	defer func() { _ = network.Free() }()
	return nil
}

func (l *LibvirtInstance) CreateDomain(id string) error {
	domainCpy := definitions.DomainXMLConfig
	domainCpy.Name = id
	domainCpy.Devices.Disks[0].Source.Volume.Volume = id
	domainCpy.Devices.Serials[0].Log.File = id

	domainXMLString, err := domainCpy.Marshal()
	if err != nil {
		return err
	}
	l.Log.Info("creating domain")
	domain, err := l.Conn.DomainCreateXML(domainXMLString, libvirt.DOMAIN_NONE)
	if err != nil {
		return fmt.Errorf("error creating libvirt domain: %s", err)
	}
	defer func() { _ = domain.Free() }()
	return nil
}

func (l *LibvirtInstance) deleteNetwork() error {
	nets, err := l.Conn.ListAllNetworks(libvirt.CONNECT_LIST_NETWORKS_ACTIVE)
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
		if name != definitions.NetworkName {
			continue
		}
		if err := net.Destroy(); err != nil {
			return err
		}
	}
	return nil
}

func (l *LibvirtInstance) deleteDomain() error {
	doms, err := l.Conn.ListAllDomains(libvirt.CONNECT_LIST_DOMAINS_ACTIVE)
	if err != nil {
		return err
	}
	defer func() {
		for _, dom := range doms {
			_ = dom.Free()
		}
	}()
	for _, dom := range doms {
		/* 		name, err := dom.GetName()
		   		if err != nil {
		   			return err
		   		}
		   		if name != definitions.DomainName {
		   			continue
		   		} */
		if err := dom.Destroy(); err != nil {
			return err
		}
	}
	return nil
}

func (l *LibvirtInstance) deleteVolume(pool *libvirt.StoragePool) error {
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
		/* 		name, err := volume.GetName()
		   		if err != nil {
		   			return err
		   		}
		   		if !strings.Contains(name, definitions.BaseDiskName) && name != definitions.BaseDiskName {
		   			continue
		   		} */
		if err := volume.Delete(libvirt.STORAGE_VOL_DELETE_NORMAL); err != nil {
			return err
		}
	}
	return nil
}

func (l *LibvirtInstance) deletePool() error {
	pools, err := l.Conn.ListAllStoragePools(libvirt.CONNECT_LIST_STORAGE_POOLS_DIR)
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
		if name != definitions.DiskPoolName {
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

func (l *LibvirtInstance) deleteLibvirtInstance() error {
	var err error
	err = multierr.Append(err, l.deleteNetwork())
	err = multierr.Append(err, l.deleteDomain())
	err = multierr.Append(err, l.deletePool())
	return err
}

func (l *LibvirtInstance) waitForQemuConnection(sig <-chan os.Signal) {
	cb := func(c *libvirt.Connect, d *libvirt.Domain, event *libvirt.DomainEventAgentLifecycle) {
		name, err := d.GetName()
		if err != nil {
			l.Log.Error("error in callback function cannot obtain name", zap.Error(err))
			return
		}
		l.Log.Info("qemu guest agent becomes ready", zap.String("name", name))
	}
	fd, err := l.Conn.DomainEventAgentLifecycleRegister(nil, cb)
	if err != nil {
		l.Log.DPanic("error getting domains", zap.Error(err))
	}
	l.Log.Info("registered Callback", zap.Int("fd", fd))
}

func (l *LibvirtInstance) InitializeBaseImagesAndNetwork() (err error) {
	// sanity check
	if err := l.deleteLibvirtInstance(); err != nil {
		return err
	}
	if err := l.CreateStoragePool(); err != nil {
		return err
	}
	if err := l.CreateBaseImage(); err != nil {
		return err
	}
	if err := l.CreateNetwork(); err != nil {
		return err
	}
	return err
}

func (l *LibvirtInstance) CreateInstance(id string) (err error) {
	if err := l.CreateBootImage("deletatio-" + id); err != nil {
		return err
	}
	if err := l.CreateDomain("deletatio-" + id); err != nil {
		return err
	}
	return nil
}

func (l *LibvirtInstance) ExecuteCommands() (err error) {
	defer func() {
		err = multierr.Append(err, l.deleteLibvirtInstance())
	}()

	for i := 0; i < 20; i++ {
		if err := l.CreateInstance(strconv.Itoa(i)); err != nil {
			return err
		}
	}

	sigs := make(chan os.Signal, 1)
	signal.Notify(sigs, syscall.SIGINT, syscall.SIGTERM)

	l.waitForQemuConnection(sigs)

	select {
	case <-sigs:
		break
	}
	l.Log.Info("termination signal received")
	signal.Stop(sigs)
	close(sigs)
	return nil
}
