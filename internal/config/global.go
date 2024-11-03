/* SPDX-License-Identifier: AGPL-3.0-only
 * Copyright (c) Benedict Schlueter
 */

package config

import (
	"embed"
	"encoding/json"
	"os"
	"time"
)

//go:embed *.pem
var rootKeyFile embed.FS

// ClusterConfiguration is the configuration for the cluster.
// Infrastructure and Kubernetes configdata is stored here.
var (
	ClusterConfiguration = ClusterConfig{
		NumberOfWorkers: 3,
		NumberOfMasters: 1,
	}
	// CleanUpTimeout is the timeout after which the save-state function is canceled when ctrl+c is pressed in the cli.
	CleanUpTimeout = 10 * time.Second
	// CiliumPath is the path to the cilium helm chart.
	CiliumPath = "https://github.com/cilium/charts/raw/master/cilium-1.16.1.tgz"
	// Cilium256Hash is the sha256 hash of the cilium helm chart.
	Cilium256Hash = "406c5bba515262c52e53b859af31af412cd8d2c332e277e98ec3e5f14382ecbf"
	// TetratePath is the path to the tetrate helm chart.
	TetratePath = "https://github.com/cilium/tetragon/releases/download/v0.8.3/tetra-linux-amd64.tar.gz"
	// Tetragon256Hash is the sha256 hash of the tetrate helm chart.
	Tetragon256Hash = "fa0e23839732cf2a3c4a1d27573431a99dd2599075bf249d3564990d906b9140"
)

const (
	// Version is the version of the project.
	Version = "0.0.1"
	// DefaultIP is the default IP address to bind to.
	DefaultIP = "0.0.0.0"
	// PublicAPIport is the port where we can access the public API.
	PublicAPIport = "9000"
	// GradeAPIport is the port where a client can request grading of the exercises.
	GradeAPIport = 9027
	// DefaultTimeout for the API.
	DefaultTimeout = 2 * time.Minute
	// AuthenticatedUserID key for a hash map, where the uid is saved.
	AuthenticatedUserID = "authenticated-uuid"
	// AuthenticationType is the type of authentication used. (i.e. pw, pk).
	AuthenticationType = "authType"
	// AuthenticatedPrivKey is the private key used for authentication.
	AuthenticatedPrivKey = "privateKey"
	// UserContainerImage is the image used for the challenge containers.
	UserContainerImage = "ghcr.io/benschlueter/delegatio/archimage:0.1"
	// SSHContainerImage is the image used for the ssh containers.
	SSHContainerImage = "ghcr.io/benschlueter/delegatio/ssh:0.1"
	// GradingContainerImage is the image used for the grading containers.
	GradingContainerImage = "ghcr.io/benschlueter/delegatio/grader:0.1"
	// UserNamespace is the namespace where the user containers are running.
	UserNamespace = "users"
	// NodeNameEnvVariable is the environment variable name of the node a user-pod is running on.
	NodeNameEnvVariable = "NODE_NAME"
	// AgentPort is the port where the agent is listening.
	AgentPort = 9000
	// SSHServiceAccountName is the name of the Kubernetes ssh service account with cluster access.
	SSHServiceAccountName = "development-ssh"
	// SSHPort is the port where the ssh server is listening.
	SSHPort = 2200
	// SSHNamespaceName is the namespace where the ssh containers are running.
	SSHNamespaceName = "ssh"
	// GraderNamespaceName is the namespace where the grader containers are running.
	GraderNamespaceName = "grader"
	// GraderServiceAccountName is the name of the Kubernetes grader service account with cluster access.
	GraderServiceAccountName = "development-grader"
	// NameSpaceFilePath is the path to the file where the namespace is stored.
	NameSpaceFilePath = "/var/run/secrets/kubernetes.io/serviceaccount/namespace"
	// SandboxPath is the path to the sandbox directory.
	SandboxPath = "/sandbox"
	// UUIDEnvVariable is the environment variable name of the uuid of the user.
	UUIDEnvVariable = "GraderUUID"
	// TerraformLogFile is the file name of the Terraform log file.
	TerraformLogFile = "terraform.log"
)

// GetExampleConfig writes an example config to config.json.
func GetExampleConfig() *UserConfiguration {
	globalConfig := UserConfiguration{
		// Currently not used
		Containers: map[string]ContainerInformation{
			"testchallenge1": {},
		},
	}

	out, err := json.Marshal(globalConfig)
	if err != nil {
		panic(err)
	}
	err = os.WriteFile("config.json", out, 0o644)
	if err != nil {
		panic(err)
	}
	return &globalConfig
}
