/* SPDX-License-Identifier: AGPL-3.0-only
 * Copyright (c) Benedict Schlueter
 */

package main

import (
	"flag"
	"log"
	"net"
	"strconv"

	"github.com/benschlueter/delegatio/internal/config"
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
	bindPort = strconv.Itoa(config.AgentPort)
	dialer := &net.Dialer{}

	run(dialer, bindIP, bindPort, zapLoggerCore)
}
