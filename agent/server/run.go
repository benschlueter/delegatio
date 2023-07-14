/* SPDX-License-Identifier: AGPL-3.0-only
 * Copyright (c) Edgeless Systems GmbH
 * Copyright (c) Benedict Schlueter
 * Copyright (c) Leonard Cohnen
 */

package main

import (
	"context"
	"net"
	"sync"

	"github.com/benschlueter/delegatio/agent/core"
	"github.com/benschlueter/delegatio/agent/core/state"
	"github.com/benschlueter/delegatio/agent/vmapi"
	"github.com/benschlueter/delegatio/agent/vmapi/vmproto"
	"github.com/benschlueter/delegatio/internal/config"
	grpc_middleware "github.com/grpc-ecosystem/go-grpc-middleware"
	grpc_zap "github.com/grpc-ecosystem/go-grpc-middleware/logging/zap"
	grpc_ctxtags "github.com/grpc-ecosystem/go-grpc-middleware/tags"
	"go.uber.org/zap"
	"google.golang.org/grpc"
	"google.golang.org/grpc/credentials/insecure"
)

var version = "0.0.0"

/*
 * This will run on the VM's bare matel. We try to contact the control plane
 * via the loadbalancerIPAddr to give us the join token. At the same time
 * we are waiting for the init-request from a user *only* if we are a control plane.
 */
func run(dialer vmapi.Dialer, bindIP, bindPort string, zapLoggerCore *zap.Logger, containerMode *bool, loadbalancerIPAddr string) {
	defer func() { _ = zapLoggerCore.Sync() }()
	zapLoggerCore.Info("starting delegatio agent", zap.String("version", version), zap.String("commit", config.Commit))

	if *containerMode {
		zapLoggerCore.Info("running in container mode")
	} else {
		zapLoggerCore.Info("running in qemu mode")
	}

	core, err := core.NewCore(zapLoggerCore, loadbalancerIPAddr)
	if err != nil {
		zapLoggerCore.Fatal("failed to create core", zap.Error(err))
	}

	vapi := vmapi.New(zapLoggerCore.Named("vmapi"), core, dialer)
	zapLoggergRPC := zapLoggerCore.Named("gRPC")

	grpcServer := grpc.NewServer(
		grpc.Creds(insecure.NewCredentials()),
		grpc.StreamInterceptor(grpc_middleware.ChainStreamServer(
			grpc_ctxtags.StreamServerInterceptor(),
			grpc_zap.StreamServerInterceptor(zapLoggergRPC),
		)),
		grpc.UnaryInterceptor(grpc_middleware.ChainUnaryServer(
			grpc_ctxtags.UnaryServerInterceptor(),
			grpc_zap.UnaryServerInterceptor(zapLoggergRPC),
		)),
	)
	vmproto.RegisterAPIServer(grpcServer, vapi)

	lis, err := net.Listen("tcp", net.JoinHostPort(bindIP, bindPort))
	if err != nil {
		zapLoggergRPC.Fatal("failed to create listener", zap.Error(err))
	}
	zapLoggergRPC.Info("server listener created", zap.String("address", lis.Addr().String()))
	core.State.Advance(state.AcceptingInit)
	go core.TryJoinCluster(context.Background())

	var wg sync.WaitGroup
	defer wg.Wait()
	wg.Add(1)
	go func() {
		defer wg.Done()
		if err := grpcServer.Serve(lis); err != nil {
			zapLoggergRPC.Fatal("failed to serve gRPC", zap.Error(err))
		}
	}()
}
