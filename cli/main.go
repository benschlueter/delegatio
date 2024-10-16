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
	flag.StringVar(&imageLocation, "path", "", "path to the image to load (required)")
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
	if err := run(ctx, log, imageLocation); err != nil {
		log.With(zap.Error(err)).DPanic("run")
	}
}
