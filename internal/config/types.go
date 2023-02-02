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
	PubKeyToUser map[string]UserInformation      `yaml:"pubkeysToUser" json:"pubkeysToUser"`
	Challenges   map[string]ChallengeInformation `yaml:"challenges" json:"challenges"`
}

// UserInformation holds the data for a user.
type UserInformation struct {
	RealName string
}

// ChallengeInformation holds the data for a challenge.
type ChallengeInformation struct{}

// KubeExecConfig holds the configuration parsed to the execCommand function.
type KubeExecConfig struct {
	Namespace     string
	PodName       string
	Command       string
	Communication ssh.Channel
	WinQueue      remotecommand.TerminalSizeQueue
	Tty           bool
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
	Namespace      string
	UserIdentifier string
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
