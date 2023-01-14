/* SPDX-License-Identifier: AGPL-3.0-only
 * Copyright (c) Benedict Schlueter
 */

package config

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
