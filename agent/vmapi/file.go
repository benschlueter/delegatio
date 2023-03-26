/* SPDX-License-Identifier: AGPL-3.0-only
 * Copyright (c) Benedict Schlueter
 */

package vmapi

import (
	"context"
	"os"
	"path/filepath"

	"github.com/benschlueter/delegatio/agent/vmapi/vmproto"
	"go.uber.org/zap"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/status"
)

// WriteFile creates a file and writes output to it.
func (a *API) WriteFile(_ context.Context, in *vmproto.WriteFileRequest) (*vmproto.WriteFileResponse, error) {
	a.logger.Info("request to write file", zap.String("path", in.Filepath), zap.String("name", in.Filename))
	if err := os.WriteFile(filepath.Join(in.Filepath, in.Filename), in.Content, os.ModeAppend); err != nil {
		a.logger.Error("failed to write file", zap.String("path", in.Filepath), zap.String("name", in.Filename), zap.Error(err))
		return nil, status.Errorf(codes.Internal, "file write failed exited with error code: %v", err)
	}
	return &vmproto.WriteFileResponse{}, nil
}

// ReadFile reads a file and returns its content.
func (a *API) ReadFile(_ context.Context, in *vmproto.ReadFileRequest) (*vmproto.ReadFileResponse, error) {
	a.logger.Info("request to read file", zap.String("path", in.Filepath), zap.String("name", in.Filename))
	content, err := os.ReadFile(filepath.Join(in.Filepath, in.Filename))
	if err != nil {
		a.logger.Error("failed to read file", zap.String("path", in.Filepath), zap.String("name", in.Filename), zap.Error(err))
		return nil, status.Errorf(codes.Internal, "file read failed exited with error code: %v", err)
	}
	return &vmproto.ReadFileResponse{Content: content}, nil
}
