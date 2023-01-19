/* SPDX-License-Identifier: AGPL-3.0-only
 * Copyright (c) Benedict Schlueter
 */

package config

import (
	"encoding/json"
	"os"
	"time"
)

// ClusterConfiguration is the configuration for the cluster.
// Infrastructure and Kubernetes configdata is stored here.
var (
	ClusterConfiguration = ClusterConfig{
		NumberOfWorkers: 2,
		NumberOfMasters: 1,
	}
	CleanUpTimeout = 10 * time.Second
	CiliumPath     = "https://github.com/cilium/charts/raw/master/cilium-1.12.5.tgz"
	CiliumHash     = "a9211be3d0d43b91aaaece1191a69c1c676a6226194bec7d806111086e620ab8"
)

const (
	// AuthenticatedUserID key for a hash map, where the sha256 fingerprint of the public key is saved.
	AuthenticatedUserID = "sha256Fingerprint"
)

// GetExampleConfig writes an example config to config.json.
func GetExampleConfig() *UserConfiguration {
	globalConfig := UserConfiguration{
		Challenges: map[string]ChallengeInformation{
			"testchallenge1": {},
			"testchallenge2": {},
			"testchallenge3": {},
		},

		PubKeyToUser: map[string]UserInformation{
			"ssh-rsa AAAAB3NzaC1yc2EAAAADAQABAAABAQDLYDO+DPlwJTKYU+S9Q1YkgC7lUJgfsq+V6VxmzdP+omp2EmEIEUsB8WFtr3kAgtAQntaCejJ9ITgoLimkoPs7bV1rA7BZZgRTL2sF+F5zJ1uXKNZz1BVeGGDDXHW5X5V/ZIlH5Bl4kNaAWGx/S5PIszkhyNXEkE6GHsSU4dz69rlutjSbwQRFLx8vjgdAxP9+jUbJMh9u5Dg1SrXiMYpzplJWFt/jI13dDlNTrhWW7790xhHur4fiQbhrVzru29BKNQtSywC+3eH2XKTzobK6h7ECS5X75ghemRIDPw32SHbQP7or1xI+MjFCrZsGyZr1L0yBFNkNAsztpWAqE2FZ": {RealName: "Benedict Schlueter"},
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
