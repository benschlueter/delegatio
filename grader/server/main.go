/* SPDX-License-Identifier: AGPL-3.0-only
 * Copyright (c) Benedict Schlueter
 */

package main

import (
	"flag"
	"fmt"
	"log"
	"net"
	"os/exec"
	"syscall"

	"github.com/benschlueter/delegatio/internal/config"
	grpc_zap "github.com/grpc-ecosystem/go-grpc-middleware/logging/zap"
	"go.uber.org/zap"
)

var userID = 555

func main() {
	var bindIP, bindPort string
	cfg := zap.NewDevelopmentConfig()

	logLevelUser := flag.Bool("debug", false, "enables gRPC debug output")
	selfExec := flag.Bool("self", false, "enables self-execution in sandbox environment")
	flag.Parse()
	args := flag.Args()

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
	bindPort = fmt.Sprint(config.GradeAPIport)
	dialer := &net.Dialer{}

	if *selfExec {
		// zaplogger prints are not visible in the final output but fmt.XXX are
		fmt.Println("EXECUTING STUFF NOW 1.0: ")
		zapLogger.Debug("self-execution enabled with args", zap.Strings("args", args))
		if err := syscall.Chroot(config.SandboxPath); err != nil {
			zapLogger.Error("chroot error", zap.Error(err))
			return
		}
		if err := syscall.Chdir("/tmp"); err != nil {
			zapLogger.Error("chdir error", zap.Error(err))
			return
		}
		if err := syscall.Mount("proc", "/proc", "proc", 0, ""); err != nil {
			fmt.Println("proc", err)
			zapLogger.Error("proc mount error", zap.Error(err))
			return
		}
		if err := syscall.Mount("sys", "/sys", "sysfs", 0, ""); err != nil {
			fmt.Println("sys", err)
			zapLogger.Error("sys mount error", zap.Error(err))
			return
		}
		if err := syscall.Mount("devpts", "/dev/pts", "devpts", 0, ""); err != nil {
			fmt.Println("devpts", err)
			zapLogger.Error("devpts mount error", zap.Error(err))
			return
		}
		if err := syscall.Setgid(userID); err != nil {
			zapLogger.Error("setgid error", zap.Error(err))
			return
		}
		if err := syscall.Setgroups([]int{userID}); err != nil {
			zapLogger.Error("setgroups error", zap.Error(err))
			return
		}
		if err := syscall.Setuid(userID); err != nil {
			zapLogger.Error("setuid error", zap.Error(err))
			return
		}
		/* 		if err := syscall.Exec(args[0], args[0:], os.Environ()); err != nil {
			fmt.Println("exec error: ", err)
		} */
		fmt.Println("EXECUTING Command")
		cmd := exec.Command(args[0], args[1:]...)
		// cmd.Run() with stdout and stderr attached does somehow not work
		output, err := cmd.Output()
		if err != nil {
			fmt.Printf("EXECUTING STUFF NOW 7.0: %v | %s\n", err, string(output))
			zapLogger.Error("exec error", zap.Error(err))
			return
		}
		fmt.Println(output)
		return
	}
	run(dialer, bindIP, bindPort, zapLoggerCore)
}
