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

	"github.com/benschlueter/delegatio/grader/gradeapi/gradeproto"
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
func New(logger *zap.Logger, dialer Dialer) (*API, error) {
	return &API{
		logger: logger,
		dialer: dialer,
	}, nil
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

func (a *API) readFile(fileName string) ([]byte, error) {
	file, err := os.Open(fileName)
	if err != nil {
		return nil, err
	}
	defer file.Close()

	if _, err := file.Seek(0, 0); err != nil {
		return nil, err
	}
	fileInfo, _ := file.Stat()
	fileSize := fileInfo.Size()
	bytes := make([]byte, fileSize)
	if _, err := file.Read(bytes); err != nil {
		return nil, err
	}
	return bytes, nil
}

// SendGradingRequest sends a grading request to the grader service.
func (a *API) SendGradingRequest(ctx context.Context, fileName string) (int, error) {
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

	_, nonceName := path.Split(f.Name())
	a.logger.Info("create nonce file", zap.String("file", nonceName))

	fileBytes, err := a.readFile(fileName)
	if err != nil {
		a.logger.Error("failed to read file", zap.String("file", fileName), zap.Error(err))
	}

	conn, err := a.dialInsecure(ctx, fmt.Sprintf("grader-service.%s.svc.cluster.local:%d", config.GraderNamespaceName, config.GradeAPIport))
	if err != nil {
		return 0, err
	}
	client := gradeproto.NewAPIClient(conn)
	resp, err := client.RequestGrading(ctx, &gradeproto.RequestGradingRequest{
		Id:       1,
		Nonce:    nonceName,
		Solution: fileBytes,
	})
	if err != nil {
		return 0, err
	}

	return int(resp.GetPoints()), nil
}
