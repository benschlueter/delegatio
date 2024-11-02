/* SPDX-License-Identifier: AGPL-3.0-only
 * Copyright (c) Benedict Schlueter
 */

package manageapi

import (
	"context"
	"net"

	"github.com/benschlueter/delegatio/agent/manageapi/manageproto"
	"go.uber.org/zap"
	"google.golang.org/grpc"
	"google.golang.org/grpc/credentials/insecure"
)

// ManageAPI is the ManageAPI.
type ManageAPI struct {
	logger *zap.Logger
	core   Core
	dialer Dialer
	manageproto.UnimplementedAPIServer
}

// New creates a new API.
func New(logger *zap.Logger, core Core, dialer Dialer) *ManageAPI {
	return &ManageAPI{
		logger: logger,
		core:   core,
		dialer: dialer,
	}
}

func (a *ManageAPI) dialInsecure(ctx context.Context, target string) (*grpc.ClientConn, error) {
	return grpc.DialContext(ctx, target,
		a.grpcWithDialer(),
		grpc.WithTransportCredentials(insecure.NewCredentials()),
		grpc.WithBlock(),
	)
}

func (a *ManageAPI) grpcWithDialer() grpc.DialOption {
	return grpc.WithContextDialer(func(ctx context.Context, addr string) (net.Conn, error) {
		return a.dialer.DialContext(ctx, "tcp", addr)
	})
}

// Dialer is the dial interface. Necessary to stub network connections for local testing
// with bufconns.
type Dialer interface {
	DialContext(ctx context.Context, network, address string) (net.Conn, error)
}
