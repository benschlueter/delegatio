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

	creds, err := createInfrastructure(ctx, log, lInstance)
	if err != nil {
		log.With(zap.Error(err)).DPanic("failed to initialize infrastructure")
	}
	log.Info("finished infrastructure initialization")
	if err := createKubernetes(ctx, log, creds, config.GetExampleConfig()); err != nil {
		log.With(zap.Error(err)).DPanic("failed to initialize kubernetes")
	}
	log.Info("finished kubernetes initialization")

	<-ctx.Done()
	<-done
}
