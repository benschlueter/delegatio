/* SPDX-License-Identifier: AGPL-3.0-only
 * Copyright (c) Benedict Schlueter
 */

package helpers

import (
	"sync"

	"github.com/benschlueter/delegatio/internal/infrastructure/utils"
	"github.com/benschlueter/delegatio/internal/store"
	"go.uber.org/zap"
	"k8s.io/client-go/kubernetes"
	"k8s.io/client-go/rest"
	"k8s.io/client-go/tools/clientcmd"
)

// Client is the struct used to access kubernetes helpers.
type Client struct {
	Client      kubernetes.Interface
	logger      *zap.Logger
	RestConfig  *rest.Config
	requestID   int
	mux         sync.Mutex
	SharedStore store.Store
}

// NewClient returns a new kuberenetes client-go wrapper.
// if no kubeconfig path is given we use the service account token.
func NewClient(logger *zap.Logger, kubeconfigPath string) (kubeClient *Client, err error) {
	// use the current context in kubeconfig
	var config *rest.Config
	if len(kubeconfigPath) == 0 {
		config, err = rest.InClusterConfig()
		if err != nil {
			return nil, err
		}
	} else {
		config, err = clientcmd.BuildConfigFromFlags("", kubeconfigPath)
		if err != nil {
			return nil, err
		}
	}
	logger.Debug("Using kubeconfig", zap.Any("config", config))
	// create the clientset
	client, err := kubernetes.NewForConfig(config)
	if err != nil {
		return nil, err
	}
	kubeClient = &Client{
		Client:     client,
		logger:     logger,
		RestConfig: config,
		requestID:  1,
		mux:        sync.Mutex{},
		// etcdClient: cli,
	}
	return
}

// ConnectToStore connects to the etcd store.
func (k *Client) ConnectToStore(creds *utils.EtcdCredentials, endpoints []string) error {
	sharedStore, err := store.NewEtcdStore(endpoints, k.logger.Named("etcd"), creds.CaCertData, creds.PeerCertData, creds.KeyData)
	if err != nil {
		return err
	}
	k.SharedStore = sharedStore
	return nil
}

