/* SPDX-License-Identifier: AGPL-3.0-only
 * Copyright (c) Benedict Schlueter
 */

package main

import (
	"flag"
	"log"
	"net"

	"cloud.google.com/go/compute/metadata"
	"github.com/benschlueter/delegatio/internal/config"
	grpc_zap "github.com/grpc-ecosystem/go-grpc-middleware/logging/zap"
	"go.uber.org/zap"
)

func main() {
	var bindIP, bindPort string
	cfg := zap.NewDevelopmentConfig()

	logLevelUser := flag.Bool("debug", false, "enables gRPC debug output")
	containerMode := flag.Bool("container", false, "signals that the agent is running in a container")
	flag.Parse()
	cfg.Level.SetLevel(zap.DebugLevel)

	zapLogger, err := cfg.Build()
	if err != nil {
		log.Fatal(err)
	}
	if *logLevelUser {
		grpc_zap.ReplaceGrpcLoggerV2(zapLogger.Named("gRPC"))
	} else {
		grpc_zap.ReplaceGrpcLoggerV2(zapLogger.WithOptions(zap.IncreaseLevel(zap.WarnLevel)).Named("gRPC"))
	}
	zapLoggerCore := zapLogger.Named("core")

	bindIP = config.DefaultIP
	bindPort = config.PublicAPIport
	dialer := &net.Dialer{}

	ipAddr, err := metadata.Get("loadbalancer")
	if err != nil {
		zapLoggerCore.Info("failed to get loadbalancer ip from metadata | not running in cloud", zap.Error(err))
	}

	run(dialer, bindIP, bindPort, zapLoggerCore, containerMode, ipAddr)
}
