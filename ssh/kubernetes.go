/* SPDX-License-Identifier: AGPL-3.0-only
 * Copyright (c) Benedict Schlueter
 */

package main

import (
	"context"
	"errors"
	"net"
	"os"
	"strings"
	"time"

	"github.com/benschlueter/delegatio/internal/installer"
	"github.com/benschlueter/delegatio/internal/store"
	"go.uber.org/zap"
)

func etcdConnector(logger *zap.Logger, client *installer.Client) (store.Store, error) {
	var err error
	var ns string
	_, err = os.Stat("./admin.conf")
	if errors.Is(err, os.ErrNotExist) {
		// ns is not ready when container spawns
		ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
		defer cancel()
		ns, err = waitForNamespaceMount(ctx)
		if err != nil {
			logger.Error("failed to get namespace, assuming default namespace \"ssh\"", zap.Error(err))
			ns = "ssh"
		}
	} else {
		ns = "ssh"
	}
	logger.Info("namespace", zap.String("namespace", ns))
	configData, err := client.Client.GetConfigMapData(context.Background(), ns, "etcd-credentials")
	if err != nil {
		return nil, err
	}
	// logger.Info("config", zap.Any("configData", configData))
	etcdStore, err := store.NewEtcdStore([]string{net.JoinHostPort(configData["advertiseAddr"], "2379")}, logger.Named("etcd"), []byte(configData["caCert"]), []byte(configData["cert"]), []byte(configData["key"]))
	if err != nil {
		return nil, err
	}
	return etcdStore, nil
}

// waitForNamespaceMount waits for the namespace file to be mounted and filled.
func waitForNamespaceMount(ctx context.Context) (string, error) {
	t := time.NewTicker(100 * time.Millisecond)
	defer t.Stop()
	for {
		select {
		case <-ctx.Done():
			return "", ctx.Err()
		case <-t.C:
			data, err := os.ReadFile("/var/run/secrets/kubernetes.io/serviceaccount/namespace")
			if err != nil && !errors.Is(err, os.ErrNotExist) {
				return "", err
			}
			ns := strings.TrimSpace(string(data))
			if len(ns) != 0 {
				return ns, nil
			}
		}
	}
}
