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
	"github.com/benschlueter/delegatio/internal/config/definitions"
	grpc_zap "github.com/grpc-ecosystem/go-grpc-middleware/logging/zap"
	"go.uber.org/zap"
)

/*
 * main is run in every docker container to allow agents to communicate with it.
 * It sets up the gRPC server and listens for incoming connections.
 * The SSH agents uses the stream exec to forward its incomming requests
 *
 * The same binary is also used in the VM to allow bootstrapping to take place via
 * CLI rpc calls.
 */
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

	var ipAddr string
	if metadata.OnGCE() {
		ipAddr, err = metadata.InstanceAttributeValue("loadbalancer")
		if err != nil {
			zapLoggerCore.Fatal("failed to get loadbalancer ip from metadata | not running in cloud", zap.Error(err))
		}

		localIP, err := metadata.InternalIP()
		if err != nil {
			zapLoggerCore.Fatal("failed to get local ip from metadata", zap.Error(err))
		}
		zapLoggerCore.Info("local ip", zap.String("ip", localIP))

		attr, err := metadata.ProjectAttributes()
		if err != nil {
			zapLoggerCore.Fatal("failed to get project attributes from metadata", zap.Error(err))
		}
		zapLoggerCore.Info("project attributes", zap.Any("attributes", attr))

		iattr, err := metadata.InstanceAttributes()
		if err != nil {
			zapLoggerCore.Fatal("failed to get instance attributes from metadata", zap.Error(err))
		}
		zapLoggerCore.Info("instance attributes", zap.Any("attributes", iattr))
	} else {
		ipAddr = definitions.NetworkXMLConfig.IPs[0].Address
	}

	run(dialer, bindIP, bindPort, zapLoggerCore, containerMode, ipAddr)
}
