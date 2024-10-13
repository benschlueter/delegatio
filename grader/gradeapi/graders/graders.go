/* SPDX-License-Identifier: AGPL-3.0-only
 * Copyright (c) Benedict Schlueter
 */

package graders

import (
	"context"
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"syscall"
	"time"

	"github.com/benschlueter/delegatio/internal/config"
	"go.uber.org/zap"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/status"
)

// Graders is responsible for maintaining state information
// of the graders. Currently we do not need any state.
type Graders struct {
	logger            *zap.Logger
	UUID              string
	singleExecTimeout time.Duration
	totalExecTimeout  time.Duration
}

// NewGraders creates and initializes a new Graders object.
func NewGraders(zapLogger *zap.Logger, studentID string) (*Graders, error) {
	c := &Graders{
		logger:            zapLogger,
		UUID:              studentID,
		singleExecTimeout: time.Second,
		totalExecTimeout:  15 * time.Second,
	}

	return c, nil
}

func (g *Graders) executeCommand(ctx context.Context, fileName string, arg ...string) ([]byte, error) {
	ctx, cancel := context.WithDeadline(ctx, time.Now().Add(g.singleExecTimeout))
	defer cancel()
	// cmd := exec.Command("/proc/self/exe", append([]string{"ns"}, os.Args[2:]...)...)
	command := exec.CommandContext(ctx, "/proc/self/exe", append([]string{"--self", fileName}, arg...)...)
	/*
	 * Create new namespaces where possible (PID, NS, NET, IPC)
	 * Mount is unnecessary as we are using chroot and all the stuff in mounted anyway
	 * We would need a stub drop process to umount all the vulnerable dirs i.e., /var
	 * A new NET / IPC namespace are created empty meaning no network and no IPC communication
	 * PID namespace is created to prevent the process from seeing other processes
	 *
	 */
	command.SysProcAttr = &syscall.SysProcAttr{
		Cloneflags:   syscall.CLONE_NEWPID | syscall.CLONE_NEWNS | syscall.CLONE_NEWNET | syscall.CLONE_NEWIPC,
		Unshareflags: syscall.CLONE_NEWNS,
	}
	// I think this here fails without an error????
	output, err := command.Output()
	if err != nil {
		g.logger.Error("failed to execute command", zap.Error(err))
		return nil, err
	}
	return output, nil
}

func (g *Graders) writeFileToDisk(_ context.Context, solution []byte) (*os.File, error) {
	f, err := os.CreateTemp(
		filepath.Join(config.SandboxPath, "tmp"),
		fmt.Sprintf("solution-%s-%v", g.UUID, "now" /*time.Now().Format(time.RFC822)*/),
	)
	if err != nil {
		g.logger.Error("failed to create content file", zap.Error(err))
		return nil, status.Error(codes.Internal, "failed to create content file")
	}
	defer f.Close()

	if _, err := f.Write(solution); err != nil {
		g.logger.Error("failed to write content file", zap.Error(err))
		return nil, err
	}
	// make executable
	if err := f.Chmod(0o777); err != nil {
		g.logger.Error("failed to chmod content file", zap.Error(err))
		return nil, err
	}
	if err := f.Sync(); err != nil {
		g.logger.Error("failed to sync content file", zap.Error(err))
		return nil, err
	}
	return f, nil
}
