/* SPDX-License-Identifier: AGPL-3.0-only
 * Copyright (c) Benedict Schlueter
 */

package graders

import (
	"context"
	"os"
	"os/exec"
	"time"

	"go.uber.org/zap"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/status"
)

// Graders is responsible for maintaining state information
// of the graders. Currently we do not need any state.
type Graders struct {
	logger            *zap.Logger
	singleExecTimeout time.Duration
	totalExecTimeout  time.Duration
}

// NewGraders creates and initializes a new Graders object.
func NewGraders(zapLogger *zap.Logger) (*Graders, error) {
	c := &Graders{
		logger:            zapLogger,
		singleExecTimeout: time.Second,
		totalExecTimeout:  15 * time.Second,
	}

	return c, nil
}

func (g *Graders) executeCommand(ctx context.Context, fileName string, arg ...string) ([]byte, error) {
	ctx, cancel := context.WithDeadline(ctx, time.Now().Add(g.singleExecTimeout))
	defer cancel()
	command := exec.CommandContext(ctx, fileName, arg...)
	output, err := command.Output()
	if err != nil {
		return nil, err
	}
	return output, nil
}

func (g *Graders) writeFileToDisk(_ context.Context, solution []byte) (*os.File, error) {
	f, err := os.CreateTemp("/tmp", "request-")
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
	if err := f.Chmod(0o700); err != nil {
		g.logger.Error("failed to chmod content file", zap.Error(err))
		return nil, err
	}
	if err := f.Sync(); err != nil {
		g.logger.Error("failed to sync content file", zap.Error(err))
		return nil, err
	}
	return f, nil
}
