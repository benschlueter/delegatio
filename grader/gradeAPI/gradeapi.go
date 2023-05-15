/* SPDX-License-Identifier: AGPL-3.0-only
 * Copyright (c) Benedict Schlueter
 */

package gradeapi

import (
	"context"
	"fmt"
	"net"
	"os"
	"path"

	"github.com/benschlueter/delegatio/grader/gradeAPI/gradeproto"
	"github.com/benschlueter/delegatio/internal/config"
	"go.uber.org/zap"
	"google.golang.org/grpc"
	"google.golang.org/grpc/credentials/insecure"
)

// API is the API.
type API struct {
	logger *zap.Logger
	dialer Dialer
	gradeproto.UnimplementedAPIServer
}

// New creates a new API.
func New(logger *zap.Logger, dialer Dialer) *API {
	return &API{
		logger: logger,
		dialer: dialer,
	}
}

// Dialer is a dialer.
type Dialer interface {
	DialContext(ctx context.Context, network, address string) (net.Conn, error)
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

// SendGradingRequest sends a grading request to the grader service.
func (a *API) SendGradingRequest(ctx context.Context) (int, error) {
	f, err := os.CreateTemp("/tmp", "gradingRequest-")
	if err != nil {
		return 0, err
	}
	defer os.Remove(f.Name())
	defer f.Close()
	// against a race condition
	if err := f.Sync(); err != nil {
		return 0, err
	}

	_, fileName := path.Split(f.Name())

	a.logger.Info("create nonce file", zap.String("file", fileName))

	conn, err := a.dialInsecure(ctx, fmt.Sprintf("grader-service.%s.svc.cluster.local:%d", config.GraderNamespaceName, config.GradeAPIport))
	if err != nil {
		return 0, err
	}
	client := gradeproto.NewAPIClient(conn)
	resp, err := client.RequestGrading(ctx, &gradeproto.RequestGradingRequest{
		Id:    1,
		Nonce: fileName,
	})
	if err != nil {
		return 0, err
	}

	return int(resp.GetPoints()), nil
}
