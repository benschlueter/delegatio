package qemu

import (
	"context"
	"strconv"
	"sync"

	"go.uber.org/zap"
	"golang.org/x/sync/errgroup"
	"libvirt.org/go/libvirt"
)

const numNodes = 3

type LibvirtInstance struct {
	ConnMux            sync.Mutex
	Conn               *libvirt.Connect
	Log                *zap.Logger
	ImagePath          string
	RegisteredDomains  map[string]*DomainInfo
	RegisteredNetworks []string
	RegisteredPools    []string
	RegisteredDisks    []string
	CancelMux          sync.Mutex
	CanelChannels      []chan struct{}
}

type DomainInfo struct {
	guestAgentReady bool
}

func (l *LibvirtInstance) ConnectWithInfrastructureService(ctx context.Context, url string) error {
	conn, err := libvirt.NewConnect(url)
	if err != nil {
		l.Log.With(zap.Error(err)).DPanic("Failed to connect to libvirt")
	}
	l.Conn = conn
	return nil
}

func (l *LibvirtInstance) InitializeInfrastructure(ctx context.Context) (err error) {
	// sanity check
	if err := l.TerminateInfrastructure(); err != nil {
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

func (l *LibvirtInstance) InitializeKubernetes(ctx context.Context, k8sConfig []byte) (err error) {
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
	l.Log.Info("network is ready")
	if err := l.blockUntilDelegatioAgentIsReady(ctx); err != nil {
		return err
	}
	l.Log.Info("delegatio-agent is ready")
	output, err := l.InitializeKubernetesgRPC(ctx, k8sConfig)
	if err != nil {
		return err
	}
	l.Log.Info("kubernetes init successful")
	if err := l.WriteKubeconfigToDisk(ctx); err != nil {
		return err
	}
	l.Log.Info("admin.conf written to disk")
	joinToken, err := l.parseKubeadmOutput(output)
	if err != nil {
		return err
	}
	kubeadmJoinToken, err := l.parseJoinCommand(joinToken)
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
