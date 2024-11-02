/* SPDX-License-Identifier: AGPL-3.0-only
 * Copyright (c) Benedict Schlueter
 */

package manageapi

import (
	"context"
	"os"
	"path/filepath"

	"github.com/benschlueter/delegatio/agent/manageapi/manageproto"
	"go.uber.org/zap"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/status"
)

// WriteFile creates a file and writes output to it.
func (a *ManageAPI) WriteFile(_ context.Context, in *manageproto.WriteFileRequest) (*manageproto.WriteFileResponse, error) {
	a.logger.Info("request to write file", zap.String("path", in.Filepath), zap.String("name", in.Filename))
	if _, err := os.Stat(in.Filepath); os.IsNotExist(err) {
		if err := os.MkdirAll(in.Filepath, 0o700); err != nil {
			a.logger.Error("failed to create directory", zap.String("path", in.Filepath), zap.Error(err))
			return nil, status.Errorf(codes.Internal, "directory creation failed exited with error code: %v", err)
		}
	}
	a.logger.Debug("about to write file to disk", zap.String("path", in.Filepath), zap.String("name", in.Filename))
	if err := os.WriteFile(filepath.Join(in.Filepath, in.Filename), in.Content, os.ModeAppend); err != nil {
		a.logger.Error("failed to write file", zap.String("path", in.Filepath), zap.String("name", in.Filename), zap.Error(err))
		return nil, status.Errorf(codes.Internal, "file write failed exited with error code: %v", err)
	}
	a.logger.Debug("wrote content to disk", zap.String("path", in.Filepath), zap.String("name", in.Filename))
	return &manageproto.WriteFileResponse{}, nil
}

// ReadFile reads a file and returns its content.
func (a *ManageAPI) ReadFile(_ context.Context, in *manageproto.ReadFileRequest) (*manageproto.ReadFileResponse, error) {
	a.logger.Info("request to read file", zap.String("path", in.Filepath), zap.String("name", in.Filename))
	content, err := os.ReadFile(filepath.Join(in.Filepath, in.Filename))
	if err != nil {
		a.logger.Error("failed to read file", zap.String("path", in.Filepath), zap.String("name", in.Filename), zap.Error(err))
		return nil, status.Errorf(codes.Internal, "file read failed exited with error code: %v", err)
	}
	return &manageproto.ReadFileResponse{Content: content}, nil
}
