/* SPDX-License-Identifier: AGPL-3.0-only
 * Copyright (c) Benedict Schlueter
 */

// code based on https://gist.github.com/protosam/53cf7970e17e06135f1622fa9955415f#file-basic-sshd-go
package main

import (
	"context"
	"os"
	"os/signal"
	"syscall"

	"github.com/benschlueter/delegatio/internal/config"
	"github.com/benschlueter/delegatio/internal/storewrapper"
	"github.com/benschlueter/delegatio/ssh/kubernetes"
	"github.com/benschlueter/delegatio/ssh/ldap"
	"go.uber.org/zap"
)

func main() {
	var client *kubernetes.K8sAPIWrapper
	var err error
	zapconf := zap.NewDevelopmentConfig()
	zapconf.Level.SetLevel(zap.DebugLevel)
	zapconf.DisableStacktrace = true
	logger, err := zapconf.Build()
	if err != nil {
		logger.With(zap.Error(err)).DPanic("Failed to create logger")
	}
	logger.Info("Starting delegatio ssh server", zap.String("commit", config.Commit))
	defer func() { _ = logger.Sync() }()

	client, err = kubernetes.NewK8sAPIWrapper(logger.Named("k8sAPI"))
	if err != nil {
		logger.With(zap.Error(err)).DPanic("failed to create k8s client")
	}

	store, err := client.GetStore()
	if err != nil {
		logger.With(zap.Error(err)).DPanic("connecting to etcd")
	}
	keys, err := storewrapper.StoreWrapper{Store: store}.GetAllKeys()
	if err != nil {
		logger.With(zap.Error(err)).DPanic("getting all keys from etcd")
	}
	logger.Debug("data in store", zap.Strings("keys", keys))
	privKey, err := storewrapper.StoreWrapper{Store: store}.GetPrivKey()
	if err != nil {
		logger.With(zap.Error(err)).DPanic("gettign priv key for ssh server")
	}
	logger.Info("pulled private key from store")
	ldap := ldap.NewLdap(logger.Named("ldap"))
	logger.Info("created ldap client")
	server := NewServer(client, logger, store, privKey, ldap)
	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	done := make(chan struct{})
	go registerSignalHandler(cancel, done, logger)
	server.Start(ctx)
	<-done
}

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
