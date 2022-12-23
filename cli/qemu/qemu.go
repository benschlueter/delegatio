package qemu

import (
	"context"
	"strconv"
	"sync"
	"time"

	"go.uber.org/zap"
	"golang.org/x/sync/errgroup"
	"libvirt.org/go/libvirt"
)

const numNodes = 3

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

func (l *LibvirtInstance) InitializeBaseImagesAndNetwork(ctx context.Context) (err error) {
	// sanity check
	if err := l.DeleteLibvirtInstance(); err != nil {
		return err
	}
	if err := l.CreateStoragePool(); err != nil {
		return err
	}
	if err := l.CreateBaseImage(ctx); err != nil {
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

func (l *LibvirtInstance) blockUntilUp(ctx context.Context) error {
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
		case <-ctx.Done():
			return ctx.Err()
		}
	}
}

func (l *LibvirtInstance) BootstrapKubernetes(ctx context.Context) (err error) {
	g, ctxGo := errgroup.WithContext(ctx)
	for i := 0; i < numNodes; i++ {
		func(id int) {
			g.Go(func() error {
				return l.CreateInstance(strconv.Itoa(id))
			})
		}(i)
	}
	if err := g.Wait(); err != nil {
		return err
	}

	if err := l.blockUntilNetworkIsReady(ctx); err != nil {
		return err
	}
	if err := l.blockUntilDelegatioAgentIsReady(ctx); err != nil {
		return err
	}
	output, err := l.InitializeKubernetesgRPC(ctx)
	if err != nil {
		return err
	}
	if err := l.WriteKubeconfigToDisk(ctx); err != nil {
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

	g, ctxGo = errgroup.WithContext(ctx)
	for i := 1; i < numNodes; i++ {
		func(id int) {
			g.Go(func() error {
				return l.JoinClustergRPC(ctxGo, "delegatio-"+strconv.Itoa(id), kubeadmJoinToken)
			})
		}(i)
	}
	if err := g.Wait(); err != nil {
		return err
	}

	return err
}
