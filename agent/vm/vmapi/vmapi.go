/* SPDX-License-Identifier: AGPL-3.0-only
 * Copyright (c) Benedict Schlueter
 */

package vmapi

import (
	"github.com/benschlueter/delegatio/agent/manageapi/manageproto"
	"github.com/benschlueter/delegatio/agent/vm/vmapi/vmproto"
	"go.uber.org/zap"
)

type vmprotoWrapper struct {
	vmproto.UnimplementedAPIServer
}

type manageprotoWrapper struct {
	manageproto.UnimplementedAPIServer
}

// APIInternal is the package internal API.
type APIInternal struct {
	logger *zap.Logger
	core   Core
	dialer Dialer
	vmprotoWrapper
	manageprotoWrapper
}

// NewInternal creates a new package internal API.
func NewInternal(logger *zap.Logger, core Core, dialer Dialer) *APIInternal {
	return &APIInternal{
		logger: logger,
		core:   core,
		dialer: dialer,
	}
}
