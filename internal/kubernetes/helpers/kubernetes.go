/* SPDX-License-Identifier: AGPL-3.0-only
 * Copyright (c) Benedict Schlueter
 */

package helpers

import (
	"errors"
	"os"
	"sync"

	"github.com/benschlueter/delegatio/internal/config"
	"github.com/benschlueter/delegatio/internal/store"
	"github.com/benschlueter/delegatio/internal/storewrapper"
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
func NewClient(logger *zap.Logger) (kubeClient *Client, err error) {
	// use the current context in kubeconfig
	var config *rest.Config
	val, present := os.LookupEnv("KUBECONFIG")
	if !present {
		config, err = rest.InClusterConfig()
		if err != nil {
			return nil, err
		}
	} else {
		config, err = clientcmd.BuildConfigFromFlags("", val)
		if err != nil {
			return nil, err
		}
	}
	logger.Info("generating kubernetes client")
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
	}
	return
}

// ConnectToStore connects to the etcd store.
func (k *Client) ConnectToStore(creds *config.EtcdCredentials, endpoints []string) error {
	if k.SharedStore != nil {
		k.logger.Info("client is already connected to store, reconnecting")
	}
	sharedStore, err := store.NewEtcdStore(endpoints, k.logger.Named("etcd"), creds.CaCertData, creds.PeerCertData, creds.KeyData)
	if err != nil {
		return err
	}
	k.SharedStore = sharedStore
	return nil
}

// GetStoreUserData saves the relevant store data to a config file.
func (k *Client) GetStoreUserData() (data *config.UserConfiguration, err error) {
	if k.SharedStore == nil {
		k.logger.Info("client is not connected to etcd")
		return nil, ErrNotConnected
	}
	stWrapper := storewrapper.StoreWrapper{Store: k.SharedStore}
	challenges, err := stWrapper.GetAllChallenges()
	if err != nil {
		return nil, err
	}
	userData, err := stWrapper.GetAllPublicKeys()
	if err != nil {
		return nil, err
	}
	data = &config.UserConfiguration{PubKeyToUser: userData, Challenges: challenges}
	return data, nil
}

// UploadSSHServerPrivKey uploads the ssh server private key to the store.
func (k *Client) UploadSSHServerPrivKey(privKey []byte) (err error) {
	if k.SharedStore == nil {
		k.logger.Info("client is not connected to etcd")
		return ErrNotConnected
	}
	stWrapper := storewrapper.StoreWrapper{Store: k.SharedStore}
	return stWrapper.PutPrivKey(privKey)
}

// DownloadSSHServerPrivKey downloads the ssh server private key from the store.
func (k *Client) DownloadSSHServerPrivKey(privKey []byte) (err error) {
	if k.SharedStore == nil {
		k.logger.Info("client is not connected to etcd")
		return ErrNotConnected
	}
	stWrapper := storewrapper.StoreWrapper{Store: k.SharedStore}
	return stWrapper.PutPrivKey(privKey)
}

// ErrNotConnected is returned when the client is not connected to etcd.
var ErrNotConnected = errors.New("client is not connected to etcd")
