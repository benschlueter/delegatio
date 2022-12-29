package main

import (
	"context"
	"errors"
	"flag"
	"os"
	"os/signal"
	"syscall"

	"github.com/benschlueter/delegatio/cli/kubernetes"
	"github.com/benschlueter/delegatio/cli/qemu"
	"github.com/benschlueter/delegatio/cli/qemu/definitions"
	"go.uber.org/zap"
	"libvirt.org/go/libvirt"
)

var version = "0.0.0"

func registerSignalHandler(cancelContext context.CancelFunc, done chan<- struct{}, log *zap.Logger) {
	sigs := make(chan os.Signal, 1)
	signal.Notify(sigs, syscall.SIGINT, syscall.SIGTERM)

	select {
	case <-sigs:
		break
	}
	log.Info("cancellation signal received")
	cancelContext()
	signal.Stop(sigs)
	close(sigs)
	done <- struct{}{}
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
	defer func() { _ = log.Sync() }()
	log.Info("starting delegatio cli", zap.String("version", version))

	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	if imageLocation == "" || imageType == "" {
		flag.Usage()
		os.Exit(1)
	}
	definitions.VolumeBootXMLConfig.BackingStore.Format.Type = imageType

	if err := libvirt.EventRegisterDefaultImpl(); err != nil {
		log.With(zap.Error(err)).DPanic("Failed to create event listener")
	}
	go func() {
		for {
			err := libvirt.EventRunDefaultImpl()
			if err != nil {
				log.With(zap.Error(err)).DPanic("go func failed")
			}
		}
	}()

	done := make(chan struct{})
	go registerSignalHandler(cancel, done, log)

	conn, err := libvirt.NewConnect("qemu:///system")
	if err != nil {
		log.With(zap.Error(err)).DPanic("Failed to connect to libvirt")
	}
	defer conn.Close()

	lInstance := qemu.NewQemu(conn, log, imageLocation)

	defer func(logger *zap.Logger, l *qemu.LibvirtInstance) {
		if err := l.DeleteLibvirtInstance(); err != nil {
			logger.Error("error while cleaning up", zap.Error(err))
		}
		log.Info("instaces terminated successfully")
	}(log, &lInstance)

	if err := lInstance.InitializeBaseImagesAndNetwork(ctx); err != nil {
		if errors.Is(err, ctx.Err()) {
			log.With(zap.Error(err)).Error("Failed to start VMs")
		} else {
			log.With(zap.Error(err)).DPanic("Failed to start VMs")
		}
	}

	if err := lInstance.BootstrapKubernetes(ctx); err != nil {
		if errors.Is(err, ctx.Err()) {
			log.With(zap.Error(err)).Error("Failed to run Kubernetes")
		} else {
			log.With(zap.Error(err)).DPanic("Failed to run Kubernetes")
		}
	}
	kClient, err := kubernetes.NewK8sClient("./admin.conf")
	if err != nil {
		if errors.Is(err, ctx.Err()) {
			log.With(zap.Error(err)).Error("Failed to connect to Kubernetes")
		} else {
			log.With(zap.Error(err)).DPanic("Failed to connect to Kubernetes")
		}
	}
	err = kClient.ListPods(ctx, "kube-system")
	if err != nil {
		if errors.Is(err, ctx.Err()) {
			log.With(zap.Error(err)).Error("Failed to list pods")
		} else {
			log.With(zap.Error(err)).DPanic("Failed to list pods")
		}
	}
	err = kClient.CreateNamespace(ctx, "testchallenge")
	if err != nil {
		if errors.Is(err, ctx.Err()) {
			log.With(zap.Error(err)).Error("Failed to create namespace")
		} else {
			log.With(zap.Error(err)).DPanic("Failed to create namespace")
		}
	}
	err = kClient.CreatePod(ctx, "testchallenge", "dummyuser")
	if err != nil {
		if errors.Is(err, ctx.Err()) {
			log.With(zap.Error(err)).Error("Failed to create namespace")
		} else {
			log.With(zap.Error(err)).DPanic("Failed to create namespace")
		}
	}

	<-ctx.Done()
	<-done
}
