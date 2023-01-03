package main

import (
	"context"
	"errors"
	"flag"
	"os"
	"os/signal"
	"syscall"
	"time"

	"github.com/benschlueter/delegatio/cli/infrastructure"
	"github.com/benschlueter/delegatio/cli/kubernetes"

	"go.uber.org/zap"
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
	err = kubeClient.ListPods(ctx, "kube-system")
	if err != nil {
		if errors.Is(err, ctx.Err()) {
			log.With(zap.Error(err)).Error("failed to list pods")
		} else {
			log.With(zap.Error(err)).DPanic("failed to list pods")
		}
	}
	err = kubeClient.CreateNamespace(ctx, "testchallenge")
	if err != nil {
		if errors.Is(err, ctx.Err()) {
			log.With(zap.Error(err)).Error("failed to create namespace")
		} else {
			log.With(zap.Error(err)).DPanic("failed to create namespace")
		}
	}
	err = kubeClient.CreateChallengeStatefulSet(ctx, "testchallenge", "dummyuser", "ssh-rsa AAAAB3NzaC1yc2EAAAADAQABAAABAQDLYDO+DPlwJTKYU+S9Q1YkgC7lUJgfsq+V6VxmzdP+omp2EmEIEUsB8WFtr3kAgtAQntaCejJ9ITgoLimkoPs7bV1rA7BZZgRTL2sF+F5zJ1uXKNZz1BVeGGDDXHW5X5V/ZIlH5Bl4kNaAWGx/S5PIszkhyNXEkE6GHsSU4dz69rlutjSbwQRFLx8vjgdAxP9+jUbJMh9u5Dg1SrXiMYpzplJWFt/jI13dDlNTrhWW7790xhHur4fiQbhrVzru29BKNQtSywC+3eH2XKTzobK6h7ECS5X75ghemRIDPw32SHbQP7or1xI+MjFCrZsGyZr1L0yBFNkNAsztpWAqE2FZ")
	if err != nil {
		if errors.Is(err, ctx.Err()) {
			log.With(zap.Error(err)).Error("failed to create statefulset")
		} else {
			log.With(zap.Error(err)).DPanic("failed to create statefulset")
		}
	}
	err = kubeClient.WaitForPodRunning(ctx, "testchallenge", "dummyuser-statefulset-0", 5*time.Minute)
	if err != nil {
		if errors.Is(err, ctx.Err()) {
			log.With(zap.Error(err)).Error("failed to wait for pod")
		} else {
			log.With(zap.Error(err)).DPanic("failed to wait for pod")
		}
	}
	/* 	err = kubeClient.CreatePodShell(ctx, "testchallenge", "dummyuser-statefulset-0", os.Stdin, os.Stdout, os.Stderr, nil)
	   	if err != nil {
	   		if errors.Is(err, ctx.Err()) {
	   			log.With(zap.Error(err)).Error("failed to spawn shell")
	   		} else {
	   			log.With(zap.Error(err)).DPanic("failed to spawn shell")
	   		}
	   	} */

	<-ctx.Done()
	<-done
}
