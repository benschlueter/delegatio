/* SPDX-License-Identifier: AGPL-3.0-only
 * Copyright (c) Benedict Schlueter
 */

// code based on https://gist.github.com/protosam/53cf7970e17e06135f1622fa9955415f#file-basic-sshd-go
package main

import (
	"context"
	"errors"
	"os"

	"github.com/benschlueter/delegatio/internal/kubernetes"
	"github.com/benschlueter/delegatio/internal/storewrapper"
	"go.uber.org/zap"
)

func main() {
	var client *kubernetes.Client
	var err error
	zapconf := zap.NewDevelopmentConfig()
	zapconf.Level.SetLevel(zap.DebugLevel)
	zapconf.DisableStacktrace = true
	logger, err := zapconf.Build()
	if err != nil {
		logger.With(zap.Error(err)).DPanic("Failed to create logger")
	}
	defer func() { _ = logger.Sync() }()
	_, err = os.Stat("./admin.conf")
	if errors.Is(err, os.ErrNotExist) {
		client, err = kubernetes.NewK8sClient(logger.Named("k8sAPI"), "")
		if err != nil {
			logger.With(zap.Error(err)).DPanic("failed to create k8s client")
		}
	} else {
		client, err = kubernetes.NewK8sClient(logger.Named("k8sAPI"), "./admin.conf")
		if err != nil {
			logger.With(zap.Error(err)).DPanic("failed to create k8s client")
		}
	}
	store, err := etcdConnector(logger, client)
	if err != nil {
		logger.With(zap.Error(err)).DPanic("connecting to etcd")
	}
	keys, err := storewrapper.StoreWrapper{Store: store}.GetAllKeys()
	if err != nil {
		logger.With(zap.Error(err)).DPanic("getting all keys from etcd")
	}
	logger.Debug("data in store", zap.Strings("keys", keys))

	server := NewSSHServer(client, logger, store)
	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	done := make(chan struct{})
	go registerSignalHandler(cancel, done, logger)
	server.StartServer(ctx)
}
