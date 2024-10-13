/* SPDX-License-Identifier: AGPL-3.0-only
 * Copyright (c) Benedict Schlueter
 */

package gradeapi

import (
	"context"
	"fmt"
	"net"

	"github.com/benschlueter/delegatio/grader/gradeapi/gradeproto"
	"github.com/benschlueter/delegatio/internal/config"
	"github.com/benschlueter/delegatio/internal/k8sapi"
	"github.com/benschlueter/delegatio/internal/store"
	"github.com/benschlueter/delegatio/internal/storewrapper"
	"go.uber.org/zap"
	"google.golang.org/grpc"
	"google.golang.org/grpc/credentials/insecure"
)

// API is the API.
type API struct {
	logger       *zap.Logger
	dialer       Dialer
	client       *k8sapi.Client
	backingStore store.Store

	gradeproto.UnimplementedAPIServer
}

// New creates a new API.
func New(logger *zap.Logger, dialer Dialer, initStore bool) (*API, error) {
	// use the current context in kubeconfig
	client, err := k8sapi.NewClient(logger)
	if err != nil {
		return nil, err
	}
	var store store.Store
	if initStore {
		store, err = client.GetStore()
		if err != nil {
			return nil, err
		}
	}

	return &API{
		logger:       logger,
		dialer:       dialer,
		client:       client,
		backingStore: store,
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

func (a *API) data() storewrapper.StoreWrapper {
	return storewrapper.StoreWrapper{Store: a.backingStore}
}

// SendGradingRequest sends a grading request to the grader service.
func (a *API) SendGradingRequest(ctx context.Context, fileBytes []byte, signature []byte, studentID string) (int, error) {
	if studentID == "" {
		return 0, fmt.Errorf("studentID is empty")
	}

	conn, err := a.dialInsecure(ctx, fmt.Sprintf("grader-service.%s.svc.cluster.local:%d", config.GraderNamespaceName, config.GradeAPIport))
	if err != nil {
		return 0, err
	}
	client := gradeproto.NewAPIClient(conn)
	resp, err := client.RequestGrading(ctx, &gradeproto.RequestGradingRequest{
		ExerciseId: 1,
		Solution:   fileBytes,
		Signature:  signature,
		StudentId:  studentID,
	})
	if err != nil {
		return 0, err
	}

	return int(resp.GetPoints()), nil
}
