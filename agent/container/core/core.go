/* SPDX-License-Identifier: AGPL-3.0-only
 * Copyright (c) Benedict Schlueter
 * Copyright (c) Edgeless Systems GmbH
 */

package core

import (
	"sync"

	"go.uber.org/zap"
)

// Core is responsible for maintaining state information of the container-agent.
type Core struct {
	zaplogger *zap.Logger
	mux       sync.Mutex
}

// NewCore creates and initializes a new Core object.
func NewCore(zapLogger *zap.Logger) (*Core, error) {
	c := &Core{
		zaplogger: zapLogger,
		mux:       sync.Mutex{},
	}
	return c, nil
}
