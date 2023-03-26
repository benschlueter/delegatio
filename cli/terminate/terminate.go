/* SPDX-License-Identifier: AGPL-3.0-only
 * Copyright (c) Benedict Schlueter
 */

package terminate

import (
	"context"
	"encoding/json"
	"net"
	"net/url"
	"os"

	"github.com/benschlueter/delegatio/internal/config"
	"github.com/benschlueter/delegatio/internal/k8sapi"

	"go.uber.org/zap"
)

// Terminate is an interface for the terminate package.
type Terminate interface {
	SaveState(context.Context, string) error
}

// terminate is a wrapper around internal kubernets.Client.
type terminate struct {
	kubeClient *k8sapi.Client
	logger     *zap.Logger
}

// NewTerminate returns a new TerminateWrapper.
func NewTerminate(logger *zap.Logger, creds *config.EtcdCredentials) (Terminate, error) {
	client, err := k8sapi.NewClient(logger)
	if err != nil {
		return nil, err
	}
	u, err := url.Parse(client.RestConfig.Host)
	if err != nil {
		return nil, err
	}
	logger.Info("endpoint", zap.String("api", u.Host))
	host, _, err := net.SplitHostPort(u.Host)
	if err != nil {
		return nil, err
	}
	if err := client.ConnectToStore(creds, []string{net.JoinHostPort(host, "2379")}); err != nil {
		return nil, err
	}
	return &terminate{kubeClient: client, logger: logger}, nil
}

func (t *terminate) SaveState(_ context.Context, configFile string) error {
	configData, err := t.kubeClient.GetStoreUserData()
	if err != nil {
		return err
	}
	byteData, err := json.Marshal(configData)
	if err != nil {
		return err
	}
	if err := os.WriteFile(configFile, byteData, 0o600); err != nil {
		return err
	}
	t.logger.Info("save state successful")
	return nil
}
