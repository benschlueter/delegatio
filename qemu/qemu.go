package qemu

import (
	"errors"
	"fmt"
	"os"
	"os/signal"
	"strconv"
	"sync"
	"syscall"
	"time"

	"go.uber.org/multierr"
	"go.uber.org/zap"
	"libvirt.org/go/libvirt"
)

const numNodes = 20

type LibvirtInstance struct {
	connMux            sync.Mutex
	conn               *libvirt.Connect
	log                *zap.Logger
	imagePath          string
	registeredDomains  map[string]*DomainInfo
	registeredNetworks []string
	registeredPools    []string
	registeredDisks    []string
	cancelMux          sync.Mutex
	canelChannels      []chan struct{}
}

type DomainInfo struct {
	guestAgentReady bool
}

func NewQemu(conn *libvirt.Connect, log *zap.Logger, imagePath string) LibvirtInstance {
	return LibvirtInstance{
		conn:              conn,
		log:               log,
		imagePath:         imagePath,
		registeredDomains: make(map[string]*DomainInfo),
	}
}

func (l *LibvirtInstance) registerQemuGuestAgentHandler() {
	cb := func(c *libvirt.Connect, d *libvirt.Domain, event *libvirt.DomainEventAgentLifecycle) {
		name, err := d.GetName()
		if err != nil {
			l.log.Error("error in callback function cannot obtain name", zap.Error(err))
			return
		}
		if event.State == 1 {
			l.connMux.Lock()
			domainState := l.registeredDomains[name]
			domainState.guestAgentReady = true
			l.connMux.Unlock()
		}
		l.log.Info("qemu guest agent changed state", zap.Any("state", event.State), zap.String("name", name))
	}
	fd, err := l.conn.DomainEventAgentLifecycleRegister(nil, cb)
	if err != nil {
		l.log.DPanic("error registering callback", zap.Error(err))
	}
	l.log.Info("registered Callback", zap.Int("fd", fd))
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
	if err := l.CreateBootImage("delegatio-" + id); err != nil {
		return err
	}
	if err := l.CreateDomain("delegatio-" + id); err != nil {
		return err
	}
	return nil
}

func (l *LibvirtInstance) blockUntilUp(stop <-chan struct{}) error {
	ticker := time.NewTicker(500 * time.Millisecond)
	defer ticker.Stop()
	for {
		select {
		case <-ticker.C:
			l.connMux.Lock()
			if val, ok := l.registeredDomains["delegatio-"+"0"]; ok {
				if val.guestAgentReady {
					l.connMux.Unlock()
					return nil
				}
			}
			l.connMux.Unlock()
		case <-stop:
			return errors.New("WaitForCompletion received stop signal")
		}
	}
}

func (l *LibvirtInstance) ExecuteCommands() (err error) {
	defer func() {
		err = multierr.Append(err, l.deleteLibvirtInstance())
	}()

	for i := 0; i < numNodes; i++ {
		go func(i int) {
			if err := l.CreateInstance(strconv.Itoa(i)); err != nil {
				l.log.Panic("error spawning qemu instances", zap.Error(err))
			}
		}(i)
	}

	sigs := make(chan os.Signal, 1)
	signal.Notify(sigs, syscall.SIGINT, syscall.SIGTERM)

	l.registerQemuGuestAgentHandler()

	// TODO: make interruptable
	if err := l.blockUntilUp(); err != nil {
		return err
	}

	output, err := l.ExecuteCommand()
	if err != nil {
		return err
	}
	joinToken, err := l.ParseKubeadmOutput(output)
	if err != nil {
		return err
	}
	kubeadmJoinToken, err := l.ParseJoinCommand(joinToken)
	if err != nil {
		return err
	}
	fmt.Println(kubeadmJoinToken)

	for i := 1; i < numNodes; i++ {
		go func(i int, l *LibvirtInstance) {
			if err := l.JoinCluster("delegatio-"+strconv.Itoa(i), kubeadmJoinToken); err != nil {
				l.log.Error("error joining cluser", zap.Error(err), zap.Int("id", i))
			}
		}(i, l)
	}

	select {
	case <-sigs:
		break
	}
	l.log.Info("termination signal received")
	for i := 0; i < numNodes; i++ {
		if domainState, ok := l.registeredDomains["delegatio-"+strconv.Itoa(i)]; ok {
			l.log.Info("domain state", zap.Bool("ready", domainState.guestAgentReady), zap.Int("number", i))
		} else {
			fmt.Println(l.registeredDomains)
		}
	}

	signal.Stop(sigs)
	close(sigs)
	return err
}
