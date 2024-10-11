/* SPDX-License-Identifier: AGPL-3.0-only
 * Copyright (c) Benedict Schlueter
 */

package main

import (
	"flag"
	"fmt"
	"log"
	"net"

	"github.com/benschlueter/delegatio/internal/config"
	grpc_zap "github.com/grpc-ecosystem/go-grpc-middleware/logging/zap"
	"go.uber.org/zap"
)

var userID = 555

func main() {
	logLevelUser := flag.Bool("debug", false, "enables gRPC debug output")
	selfExec := flag.Bool("self", false, "enables self-execution in sandbox environment")
	flag.Parse()
	args := flag.Args()

	if *selfExec {
		runSelfExec(args, userID)
	} else {
		cfg := zap.NewDevelopmentConfig()
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
		bindIP := config.DefaultIP
		bindPort := fmt.Sprint(config.GradeAPIport)
		dialer := &net.Dialer{}
		run(dialer, bindIP, bindPort, zapLoggerCore)
	}
}
