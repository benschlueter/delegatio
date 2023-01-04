package main

import (
	"context"
	"errors"
	"flag"
	"os"
	"os/signal"
	"syscall"

	"github.com/benschlueter/delegatio/cli/infrastructure"
	"github.com/benschlueter/delegatio/cli/kubernetes"

	"go.uber.org/zap"
)

var version = "0.0.0"

func registerSignalHandler(cancelContext context.CancelFunc, done chan<- struct{}, log *zap.Logger) {
	sigs := make(chan os.Signal, 1)
	signal.Notify(sigs, syscall.SIGINT, syscall.SIGTERM)

	<-sigs

	log.Info("cancellation signal received")
	cancelContext()
	signal.Stop(sigs)
	close(sigs)
	done <- struct{}{}
}

func main() {
	var imageLocation string
	flag.StringVar(&imageLocation, "path", "", "path to the image to measure (required)")
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

	if imageLocation == "" {
		flag.Usage()
		os.Exit(1)
	}

	done := make(chan struct{})
	go registerSignalHandler(cancel, done, log)

	lInstance := infrastructure.NewQemu(log.Named("infra"), imageLocation)

	defer func(logger *zap.Logger, l infrastructure.Infrastructure) {
		if err := l.TerminateInfrastructure(); err != nil {
			logger.Error("error while cleaning up", zap.Error(err))
		}
		log.Info("instaces terminated successfully")
		if err := l.TerminateConnection(); err != nil {
			logger.Error("error while cleaning up", zap.Error(err))
		}
		log.Info("connection successfully closed")
	}(log, lInstance)

	if err := lInstance.ConnectWithInfrastructureService(ctx, "qemu:///system"); err != nil {
		if errors.Is(err, ctx.Err()) {
			log.With(zap.Error(err)).Error("failed to connect to infrastructure service")
		} else {
			log.With(zap.Error(err)).DPanic("failed to connect to infrastructure service")
		}
	}

	if err := lInstance.InitializeInfrastructure(ctx); err != nil {
		if errors.Is(err, ctx.Err()) {
			log.With(zap.Error(err)).Error("failed to start VMs")
		} else {
			log.With(zap.Error(err)).DPanic("failed to start VMs")
		}
	}

	kubeConf, err := infrastructure.GetKubeInitConfig()
	if err != nil {
		log.With(zap.Error(err)).DPanic("failed to get kubeConfig")
	}

	if err := lInstance.InitializeKubernetes(ctx, kubeConf); err != nil {
		if errors.Is(err, ctx.Err()) {
			log.With(zap.Error(err)).Error("failed to run Kubernetes")
		} else {
			log.With(zap.Error(err)).DPanic("failed to run Kubernetes")
		}
	}
	kubeClient, err := kubernetes.NewK8sClient("./admin.conf", log.Named("k8sAPI"))
	if err != nil {
		if errors.Is(err, ctx.Err()) {
			log.With(zap.Error(err)).Error("failed to connect to Kubernetes")
		} else {
			log.With(zap.Error(err)).DPanic("failed to connect to Kubernetes")
		}
	}
	err = kubeClient.InstallHelmStuff(ctx)
	if err != nil {
		if errors.Is(err, ctx.Err()) {
			log.With(zap.Error(err)).Error("failed to install helm charts")
		} else {
			log.With(zap.Error(err)).DPanic("failed to install helm charts")
		}
	}
	if err := kubeClient.CreatePersistentVolume(ctx, "root-storage-claim", "testchallenge1"); err != nil {
		if errors.Is(err, ctx.Err()) {
			log.With(zap.Error(err)).Error("failed to install helm charts")
		} else {
			log.With(zap.Error(err)).DPanic("failed to install helm charts")
		}
	}
	if err := kubeClient.CreatePersistentVolumeClaim(ctx, "root-storage-claim", "testchallenge1"); err != nil {
		if errors.Is(err, ctx.Err()) {
			log.With(zap.Error(err)).Error("failed to install helm charts")
		} else {
			log.With(zap.Error(err)).DPanic("failed to install helm charts")
		}
	}

	<-ctx.Done()
	<-done
}
