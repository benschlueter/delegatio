/* SPDX-License-Identifier: AGPL-3.0-only
 * Copyright (c) Benedict Schlueter
 */

package config

import (
	"golang.org/x/crypto/ssh"
	"k8s.io/client-go/tools/remotecommand"
)

// ClusterConfig is the configuration for the cluster.
type ClusterConfig struct {
	NumberOfWorkers int
	NumberOfMasters int
}

// UserConfiguration is the configuration for the user.
type UserConfiguration struct {
	UUIDToUser   map[string]UserInformation      `yaml:"uuidToUser" json:"uuidToUser"`
	PubKeyToUser map[string]UserInformation      `yaml:"pubkeysToUser" json:"pubkeysToUser"`
	Containers   map[string]ContainerInformation `yaml:"challenges" json:"challenges"`
}

// UserInformation holds the data for a user.
type UserInformation struct {
	Username   string
	RealName   string
	Email      string
	LegiNumber string
	UUID       string
	Gender     string
	PrivKey    []byte
	PubKey     []byte
	Points     map[string]int
}

// ContainerInformation holds the data for a challenge.
type ContainerInformation struct {
	ContainerName string
}

// KubeExecConfig holds the configuration parsed to the execCommand function.
type KubeExecConfig struct {
	Namespace      string
	UserIdentifier string
	Command        string
	Communication  ssh.Channel
	WinQueue       remotecommand.TerminalSizeQueue
	Tty            bool
}

// KubeFileWriteConfig holds the data to write a file using the VMAPI.
type KubeFileWriteConfig struct {
	UserIdentifier string
	Namespace      string
	FileName       string
	FilePath       string
	FileData       []byte
}

// KubeForwardConfig holds the configuration parsed to the forwardCommand function.
type KubeForwardConfig struct {
	Namespace     string
	PodName       string
	Port          string
	Communication ssh.Channel
}

// KubeRessourceIdentifier holds the information to identify a kubernetes ressource.
type KubeRessourceIdentifier struct {
	UserIdentifier      string
	Namespace           string
	ContainerIdentifier string
	NodeName            string
	StorageClass        string
}

// EtcdCredentials contains the credentials for etcd.
type EtcdCredentials struct {
	PeerCertData []byte // self generated
	KeyData      []byte // self generated
	CaCertData   []byte // "/etc/kubernetes/pki/etcd/ca.crt"
}

// NodeInformation contains the information about the nodes.
type NodeInformation struct {
	Masters map[string]string
	Workers map[string]string
}
