/* SPDX-License-Identifier: AGPL-3.0-only
 * Copyright (c) Benedict Schlueter
 */

package config

import (
	"encoding/json"
	"os"
)

// UserConfiguration is the configuration for the user.
type UserConfiguration struct {
	PubKeyToUser map[string]string   `yaml:"pubkeysToUser" json:"pubkeysToUser"`
	Challenges   map[string]struct{} `yaml:"challenges" json:"challenges"`
}

// GetExampleConfig writes an example config to config.json.
func GetExampleConfig() *UserConfiguration {
	globalConfig := UserConfiguration{
		Challenges: map[string]struct{}{
			"testchallenge1": {},
			"testchallenge2": {},
			"testchallenge3": {},
		},

		PubKeyToUser: map[string]string{
			"ssh-rsa AAAAB3NzaC1yc2EAAAADAQABAAABAQDLYDO+DPlwJTKYU+S9Q1YkgC7lUJgfsq+V6VxmzdP+omp2EmEIEUsB8WFtr3kAgtAQntaCejJ9ITgoLimkoPs7bV1rA7BZZgRTL2sF+F5zJ1uXKNZz1BVeGGDDXHW5X5V/ZIlH5Bl4kNaAWGx/S5PIszkhyNXEkE6GHsSU4dz69rlutjSbwQRFLx8vjgdAxP9+jUbJMh9u5Dg1SrXiMYpzplJWFt/jI13dDlNTrhWW7790xhHur4fiQbhrVzru29BKNQtSywC+3eH2XKTzobK6h7ECS5X75ghemRIDPw32SHbQP7or1xI+MjFCrZsGyZr1L0yBFNkNAsztpWAqE2FZ": "Benedict Schlueter",
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