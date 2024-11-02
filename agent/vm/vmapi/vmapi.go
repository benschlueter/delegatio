/* SPDX-License-Identifier: AGPL-3.0-only
 * Copyright (c) Benedict Schlueter
 */

package vmapi

import (
	"context"
	"net"

	"github.com/benschlueter/delegatio/agent/manageapi/manageproto"
	"github.com/benschlueter/delegatio/agent/vm/vmapi/vmproto"
	"go.uber.org/zap"
	"google.golang.org/grpc"
	"google.golang.org/grpc/credentials/insecure"
)

type vmprotoWrapper struct {
	vmproto.UnimplementedAPIServer
}

type manageprotoWrapper struct {
	manageproto.UnimplementedAPIServer
}

// API is the API.
type API struct {
	logger *zap.Logger
	core   Core
	dialer Dialer
	vmprotoWrapper
	manageprotoWrapper
}

// New creates a new API.
func New(logger *zap.Logger, core Core, dialer Dialer) *API {
	return &API{
		logger: logger,
		core:   core,
		dialer: dialer,
	}
}

func (a *API) dialInsecure(ctx context.Context, target string) (*grpc.ClientConn, error) {
	return grpc.DialContext(ctx, target,
		a.grpcWithDialer(),
		grpc.WithTransportCredentials(insecure.NewCredentials()),
		grpc.WithBlock(),
	)
}

func (a *API) grpcWithDialer() grpc.DialOption {
	return grpc.WithContextDialer(func(ctx context.Context, addr string) (net.Conn, error) {
		return a.dialer.DialContext(ctx, "tcp", addr)
	})
}

// Dialer is the dial interface. Necessary to stub network connections for local testing
// with bufconns.
type Dialer interface {
	DialContext(ctx context.Context, network, address string) (net.Conn, error)
}

// ToDo: Put Kubernetes functions here to make use of the dial functions?
