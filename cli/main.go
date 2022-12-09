package main

import (
	"context"
	"flag"
	"os"
	"os/signal"
	"syscall"

	"github.com/benschlueter/delegatio/qemu"
	"github.com/benschlueter/delegatio/qemu/definitions"
	"go.uber.org/zap"
	"libvirt.org/go/libvirt"
)

var version = "0.0.0"

func registerSignalHandler(cancel context.CancelFunc, log *zap.Logger) {
	sigs := make(chan os.Signal, 1)
	signal.Notify(sigs, syscall.SIGINT, syscall.SIGTERM)

	select {
	case <-sigs:
		break
	}
	log.Info("cancellation signal received")
	cancel()
	signal.Stop(sigs)
	close(sigs)
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

	go registerSignalHandler(cancel, log)

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
		log.With(zap.Error(err)).DPanic("Failed to initialize Libvirt")
	}

	if err := lInstance.BootstrapKubernetes(ctx); err != nil {
		log.With(zap.Error(err)).DPanic("Failed to run VMs")
	}

	select {
	case <-ctx.Done():
	}
}
