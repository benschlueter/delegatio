/* SPDX-License-Identifier: AGPL-3.0-only
 * Copyright (c) Edgeless Systems GmbH
 * Copyright (c) Benedict Schlueter
 * Copyright (c) Leonard Cohnen
 */

package main

import (
	"fmt"
	"log"
	"net"
	"os"
	"os/exec"
	"os/signal"
	"sync"
	"syscall"

	"github.com/benschlueter/delegatio/grader/gradeapi"
	"github.com/benschlueter/delegatio/grader/gradeapi/gradeproto"
	"github.com/benschlueter/delegatio/internal/config"
	grpc_middleware "github.com/grpc-ecosystem/go-grpc-middleware"
	grpc_zap "github.com/grpc-ecosystem/go-grpc-middleware/logging/zap"
	grpc_ctxtags "github.com/grpc-ecosystem/go-grpc-middleware/tags"
	"go.uber.org/zap"
	"golang.org/x/sys/unix"
	"google.golang.org/grpc"
	"google.golang.org/grpc/credentials/insecure"
)

var version = "0.0.0"

func run(dialer gradeapi.Dialer, bindIP, bindPort string, zapLoggerCore *zap.Logger) {
	defer func() { _ = zapLoggerCore.Sync() }()
	zapLoggerCore.Info("starting delegatio grader", zap.String("version", version), zap.String("commit", config.Commit))
	gapi, err := gradeapi.New(zapLoggerCore.Named("gradeapi"), dialer, true)
	if err != nil {
		zapLoggerCore.Fatal("create gradeapi", zap.Error(err))
	}

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
	gradeproto.RegisterAPIServer(grpcServer, gapi)

	lis, err := net.Listen("tcp", net.JoinHostPort(bindIP, bindPort))
	if err != nil {
		zapLoggergRPC.Fatal("create listener", zap.Error(err))
	}
	zapLoggergRPC.Info("server listener created", zap.String("address", lis.Addr().String()))

	if err := setupDevMount(zapLoggerCore); err != nil {
		zapLoggerCore.Fatal("setup dev mount", zap.Error(err))
	}
	done := make(chan struct{})
	go registerSignalHandler(done, zapLoggerCore)

	var wg sync.WaitGroup
	defer wg.Wait()
	wg.Add(1)
	go func() {
		defer wg.Done()
		if err := grpcServer.Serve(lis); err != nil {
			zapLoggergRPC.Fatal("serve gRPC", zap.Error(err))
		}
	}()
	<-done
	if err := setupDevUmount(zapLoggerCore); err != nil {
		zapLoggerCore.Fatal("umount dev", zap.Error(err))
	}
}

func runSelfExec(args []string, userID int) {
	// zaplogger prints are not visible in the final output but fmt.XXX are
	// maybe because zaplogger had to deal with different stdout?
	if err := syscall.Chroot(config.SandboxPath); err != nil {
		log.Fatalf("chroot: %v", zap.Error(err))
		return
	}
	if err := syscall.Mount("proc", "/proc", "proc", 0, ""); err != nil {
		log.Fatalf("proc mount: %v", zap.Error(err))
		return
	}
	defer func() {
		_ = syscall.Unmount("/proc", 0)
	}()
	if err := syscall.Mount("sys", "/sys", "sysfs", 0, ""); err != nil {
		log.Fatalf("sys mount: %v", zap.Error(err))
		return
	}
	defer func() {
		_ = syscall.Unmount("/sys", 0)
	}()
	if err := syscall.Mount("devpts", "/dev/pts", "devpts", 0, ""); err != nil {
		log.Fatalf("devpts mount: %v", zap.Error(err))
		return
	}
	defer func() {
		_ = syscall.Unmount("/dev/pts", 0)
	}()
	if err := syscall.Setgid(userID); err != nil {
		log.Fatalf("setgid: %v", zap.Error(err))
		return
	}
	if err := syscall.Setgroups([]int{userID}); err != nil {
		log.Fatalf("setgroups: %v", zap.Error(err))
		return
	}
	if err := syscall.Setuid(userID); err != nil {
		log.Fatalf("setuid: %v", zap.Error(err))
		return
	}
	cmd := exec.Command(args[0], args[1:]...)
	output, err := cmd.Output()
	if err != nil {
		log.Fatalf("exec: %v", zap.Error(err))
		return
	}
	fmt.Println(string(output))
	return
}

func setupDevMount(zapLogger *zap.Logger) error {
	// Mount the /tmp directory
	flags := uintptr(unix.MS_BIND | unix.MS_REC)
	if err := syscall.Mount("/dev", fmt.Sprintf("%s/dev", config.SandboxPath), "devtmpfs", flags, ""); err != nil {
		zapLogger.Error("dev mount error", zap.Error(err))
		return err
	}
	if err := syscall.Mount("/tmp", fmt.Sprintf("%s/tmp", config.SandboxPath), "tmpfs", flags, ""); err != nil {
		zapLogger.Error("tmp mount error", zap.Error(err))
		return err
	}
	return nil
}

func setupDevUmount(zapLogger *zap.Logger) error {
	// Mount the /tmp directory
	if err := syscall.Unmount(fmt.Sprintf("%s/dev", config.SandboxPath), 0); err != nil {
		zapLogger.Error("dev mount error", zap.Error(err))
		return err
	}
	if err := syscall.Unmount(fmt.Sprintf("%s/tmp", config.SandboxPath), 0); err != nil {
		zapLogger.Error("tmp mount error", zap.Error(err))
		return err
	}
	return nil
}

func registerSignalHandler(done chan<- struct{}, log *zap.Logger) {
	sigs := make(chan os.Signal, 1)
	signal.Notify(sigs, syscall.SIGINT, syscall.SIGTERM)

	<-sigs

	log.Info("cancellation signal received")
	signal.Stop(sigs)
	close(sigs)
	done <- struct{}{}
}
