/* SPDX-License-Identifier: AGPL-3.0-only
 * Copyright (c) Benedict Schlueter
 */

package main

import (
	"context"
	"flag"
	"os"
	"os/signal"
	"syscall"

	"github.com/benschlueter/delegatio/internal/config"
	"github.com/benschlueter/delegatio/internal/infrastructure"

	"go.uber.org/zap"
)

var version = "0.0.0"

func registerSignalHandler(ctx context.Context, log *zap.Logger) (context.Context, context.CancelFunc) {
	ctx, cancelFunc := signal.NotifyContext(ctx, syscall.SIGINT, syscall.SIGTERM)
	stopped := make(chan struct{}, 1)
	done := make(chan struct{}, 1)

	go func() {
		defer func() {
			cancelFunc()
			stopped <- struct{}{}
		}()
		select {
		case <-ctx.Done():
			log.Info("ctrl+c caught, stopping gracefully")
		case <-done:
			log.Info("done signal received, stopping gracefully")
		}
	}()

	return ctx, func() {
		done <- struct{}{}
		<-stopped
	}
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
	log.Info("starting delegatio cli", zap.String("version", version), zap.String("commit", config.Commit))
	ctx, cancel := registerSignalHandler(context.Background(), log)
	defer cancel()

	if imageLocation == "" {
		flag.Usage()
		os.Exit(1)
	}

	lInstance, err := infrastructure.NewQemu(log.Named("infra"), imageLocation)
	if err != nil {
		log.With(zap.Error(err)).DPanic("error creating infrastructure")
	}

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

	creds, err := createInfrastructure(ctx, log, lInstance)
	if err != nil {
		log.With(zap.Error(err)).DPanic("create infrastructure")
	}
	log.Info("finished infrastructure initialization")
	kubewrapper, err := NewKubeWrapper(log.Named("kubeWrapper"))
	if err != nil {
		log.With(zap.Error(err)).DPanic("new kubeWrapper")
	}
	if err := kubewrapper.createKubernetes(ctx, creds, config.GetExampleConfig()); err != nil {
		log.With(zap.Error(err)).DPanic("failed to initialize kubernetes")
	}
	log.Info("finished kubernetes initialization")

	<-ctx.Done()
	cleanUpCtx, secondCancel := context.WithTimeout(context.Background(), config.CleanUpTimeout)
	defer secondCancel()
	if err := kubewrapper.saveKubernetesState(cleanUpCtx, "./kubernetes-state.json"); err != nil {
		log.Error("failed to save kubernetes state", zap.Error(err))
	}
}
