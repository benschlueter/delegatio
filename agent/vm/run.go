/* SPDX-License-Identifier: AGPL-3.0-only
 * Copyright (c) Edgeless Systems GmbH
 * Copyright (c) Benedict Schlueter
 * Copyright (c) Leonard Cohnen
 */

package main

import (
	"context"
	"net"
	"os/signal"
	"syscall"

	"github.com/benschlueter/delegatio/agent/manageapi"
	"github.com/benschlueter/delegatio/agent/manageapi/manageproto"
	"github.com/benschlueter/delegatio/agent/vm/core"
	"github.com/benschlueter/delegatio/agent/vm/core/state"
	"github.com/benschlueter/delegatio/agent/vm/vmapi"
	"github.com/benschlueter/delegatio/agent/vm/vmapi/vmproto"
	"github.com/benschlueter/delegatio/internal/config"
	grpc_middleware "github.com/grpc-ecosystem/go-grpc-middleware"
	grpc_zap "github.com/grpc-ecosystem/go-grpc-middleware/logging/zap"
	grpc_ctxtags "github.com/grpc-ecosystem/go-grpc-middleware/tags"
	"go.uber.org/zap"
	"golang.org/x/sync/errgroup"
	"google.golang.org/grpc"
	"google.golang.org/grpc/credentials"
)

var version = "0.0.0"

/*
 * This will run on the VM's bare matel. We try to contact the control plane
 * via the loadbalancerIPAddr to give us the join token. At the same time
 * we are waiting for the init-request from a user *only* if we are a control plane.
 */
func run(dialer vmapi.Dialer, bindIP, agentPort, vmPort string, zapLoggerCore *zap.Logger, containerMode *bool, loadbalancerIPAddr string) {
	defer func() { _ = zapLoggerCore.Sync() }()
	zapLoggerCore.Info("starting delegatio agent", zap.String("version", version), zap.String("commit", config.Commit))

	if *containerMode {
		zapLoggerCore.Info("running in container mode")
	} else {
		zapLoggerCore.Info("running in qemu mode")
	}
	vmapiExternal, err := vmapi.NewExternal(zapLoggerCore.Named("vmapi"), &net.Dialer{})
	if err != nil {
		zapLoggerCore.Fatal("create vmapi external", zap.Error(err))
	}
	core, err := core.NewCore(zapLoggerCore, loadbalancerIPAddr, vmapiExternal)
	if err != nil {
		zapLoggerCore.Fatal("create core", zap.Error(err))
	}

	vapi := vmapi.NewInternal(zapLoggerCore.Named("vmapi"), core, dialer)
	mapi := manageapi.New(zapLoggerCore.Named("manageapi"), core, dialer)
	zapLoggergRPC := zapLoggerCore.Named("gRPC")

	tlsConfig, err := config.GenerateTLSConfigServer()
	if err != nil {
		zapLoggerCore.Fatal("generate TLS config", zap.Error(err))
	}
	zapLoggerCore.Info("TLS config generated")

	grpcServerAgent := grpc.NewServer(
		grpc.Creds(credentials.NewTLS(tlsConfig)),
		grpc.StreamInterceptor(grpc_middleware.ChainStreamServer(
			grpc_ctxtags.StreamServerInterceptor(),
			grpc_zap.StreamServerInterceptor(zapLoggergRPC),
		)),
		grpc.UnaryInterceptor(grpc_middleware.ChainUnaryServer(
			grpc_ctxtags.UnaryServerInterceptor(),
			grpc_zap.UnaryServerInterceptor(zapLoggergRPC),
		)),
	)

	grpcServerVM := grpc.NewServer(
		grpc.Creds(credentials.NewTLS(tlsConfig)),
		grpc.StreamInterceptor(grpc_middleware.ChainStreamServer(
			grpc_ctxtags.StreamServerInterceptor(),
			grpc_zap.StreamServerInterceptor(zapLoggergRPC),
		)),
		grpc.UnaryInterceptor(grpc_middleware.ChainUnaryServer(
			grpc_ctxtags.UnaryServerInterceptor(),
			grpc_zap.UnaryServerInterceptor(zapLoggergRPC),
		)),
	)

	vmproto.RegisterAPIServer(grpcServerVM, vapi)

	manageproto.RegisterAPIServer(grpcServerAgent, mapi)

	lisAgent, err := net.Listen("tcp", net.JoinHostPort(bindIP, agentPort))
	if err != nil {
		zapLoggergRPC.Fatal("failed to create listener", zap.Error(err))
	}
	zapLoggergRPC.Info("server listener created", zap.String("address", lisAgent.Addr().String()))
	lisVM, err := net.Listen("tcp", net.JoinHostPort(bindIP, vmPort))
	if err != nil {
		zapLoggergRPC.Fatal("failed to create listener", zap.Error(err))
	}
	zapLoggergRPC.Info("server listener created", zap.String("address", lisAgent.Addr().String()))

	core.State.Advance(state.AcceptingInit)

	ctx, cancel := registerSignalHandler(context.Background(), zapLoggerCore)
	defer cancel()
	g, _ := errgroup.WithContext(ctx)
	g.Go(func() error {
		return grpcServerAgent.Serve(lisAgent)
	})
	g.Go(func() error {
		return grpcServerVM.Serve(lisVM)
	})
	g.Go(func() error {
		return core.TryJoinCluster(context.Background())
	})

	if err := g.Wait(); err != nil {
		zapLoggergRPC.Fatal("server error", zap.Error(err))
	}
}

func registerSignalHandler(ctx context.Context, log *zap.Logger) (context.Context, context.CancelFunc) {
	ctx, cancelFunc := signal.NotifyContext(ctx, syscall.SIGINT, syscall.SIGTERM)
	stopped := make(chan struct{}, 1)
	done := make(chan struct{}, 1)

	go func() {
		defer func() {
			cancelFunc()
			stopped <- struct{}{}
		}()
		select {
		case <-ctx.Done():
			log.Info("ctrl+c caught, stopping gracefully")
		case <-done:
			log.Info("done signal received, stopping gracefully")
		}
	}()

	return ctx, func() {
		done <- struct{}{}
		<-stopped
	}
}
