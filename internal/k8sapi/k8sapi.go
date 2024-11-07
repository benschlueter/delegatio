/* SPDX-License-Identifier: AGPL-3.0-only
 * Copyright (c) Benedict Schlueter
 */

package k8sapi

import (
	"context"
	"errors"
	"net"
	"os"
	"strings"
	"sync"
	"time"

	"github.com/benschlueter/delegatio/internal/config"
	"github.com/benschlueter/delegatio/internal/store"
	"github.com/benschlueter/delegatio/internal/storewrapper"
	"go.uber.org/zap"
	"k8s.io/client-go/kubernetes"
	"k8s.io/client-go/rest"
	"k8s.io/client-go/tools/clientcmd"
)

// TODO: When functions are stable expose them through an interface.

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

// ConnectToStoreExternal connects to the etcd store.
func (k *Client) ConnectToStoreExternal(creds *config.EtcdCredentials, endpoints []string) error {
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
	data = &config.UserConfiguration{UUIDToUser: userData, Containers: challenges}
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
func (k *Client) DownloadSSHServerPrivKey() ([]byte, error) {
	if k.SharedStore == nil {
		k.logger.Info("client is not connected to etcd")
		return nil, ErrNotConnected
	}
	stWrapper := storewrapper.StoreWrapper{Store: k.SharedStore}
	return stWrapper.GetPrivKey()
}

// GetKubeConfigPath returns the path to the kubeconfig file.
func GetKubeConfigPath() (string, error) {
	val, present := os.LookupEnv("KUBECONFIG")
	if !present {
		return "", errors.New("KUBECONFIG not set")
	}
	if _, err := os.Stat(val); err != nil {
		if os.IsNotExist(err) {
			return "", errors.New("KUBECONFIG file does not exist")
		}
		return "", err
	}
	return val, nil
}

// GetStore returns a store backed by kube etcd. Its only supposed to used within a kubernetes pod.
func (k *Client) GetStore() (store.Store, error) {
	if k.SharedStore == nil {
		if err := k.ConnectToStoreInternal(); err != nil {
			k.logger.Info("client is not connected to etcd")
			return nil, err
		}
	}
	return k.SharedStore, nil
}

// Before calling this the installer needs to populate a config map with the respective credentials.
// Furthermore a serviceaccount must be set up for the namespace and it needs to be attached to the
// running pod.
func (k *Client) ConnectToStoreInternal() error {
	var err error
	var ns string
	if _, err := os.Stat(config.NameSpaceFilePath); errors.Is(err, os.ErrNotExist) {
		// ns is not ready when container spawns
		ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
		defer cancel()
		ns, err = waitForNamespaceMount(ctx)
		if err != nil {
			k.logger.Error("failed to get namespace after timeout", zap.Error(err))
			return err
		}
	} else {
		// out of cluster mode currently assumes 'ssh' namespace
		if content, err := os.ReadFile(config.NameSpaceFilePath); err == nil {
			ns = strings.TrimSpace(string(content))
		} else {
			return err
		}
	}
	k.logger.Info("namespace", zap.String("namespace", ns))
	configData, err := k.GetConfigMapData(context.Background(), ns, "etcd-credentials")
	if err != nil {
		return err
	}
	// logger.Info("config", zap.Any("configData", configData))
	etcdStore, err := store.NewEtcdStore([]string{net.JoinHostPort(configData["advertiseAddr"], "2379")}, k.logger, []byte(configData["caCert"]), []byte(configData["cert"]), []byte(configData["key"]))
	if err != nil {
		return err
	}
	k.SharedStore = etcdStore
	return nil
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

// ErrNotConnected is returned when the client is not connected to etcd.
var ErrNotConnected = errors.New("client is not connected to etcd")
