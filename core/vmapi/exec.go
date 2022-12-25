package vmapi

import (
	"bytes"
	"context"
	"io"
	"os"
	"os/exec"
	"path/filepath"

	"github.com/benschlueter/delegatio/core/vmapi/vmproto"
	"go.uber.org/zap"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/status"
)

type streamWriter struct {
	forward func([]byte) error
}

func (sw streamWriter) Write(p []byte) (int, error) {
	if err := sw.forward(p); err != nil {
		return 0, err
	}
	return len(p), nil
}

func (a *API) ExecCommandStream(in *vmproto.ExecCommandStreamRequest, srv vmproto.API_ExecCommandStreamServer) error {
	a.logger.Info("request to execute command", zap.String("command", in.Command), zap.Strings("args", in.Args))
	command := exec.Command(in.Command, in.Args...)
	streamer := streamWriter{forward: func(b []byte) error {
		return srv.Send(&vmproto.ExecCommandStreamResponse{
			Content: &vmproto.ExecCommandStreamResponse_Log{
				Log: &vmproto.Log{
					Message: string(b),
				},
			},
		})
	}}
	var stdoutBuf, stderrBuf bytes.Buffer

	command.Stdout = io.MultiWriter(streamer, &stdoutBuf)
	command.Stderr = io.MultiWriter(streamer, &stderrBuf)

	err := command.Start()
	if err != nil {
		return status.Errorf(codes.Internal, "command exited with error code: %v and output: %s", err, stdoutBuf.Bytes())
	}

	if err := command.Wait(); err != nil {
		return status.Errorf(codes.Internal, "command exited with error code: %v and output: %s", err, stdoutBuf.Bytes())
	}
	return srv.Send(&vmproto.ExecCommandStreamResponse{Content: &vmproto.ExecCommandStreamResponse_Output{Output: stdoutBuf.Bytes()}})
}

// ActivateAdditionalNodes is the RPC call to activate additional nodes.
func (a *API) ExecCommand(ctx context.Context, in *vmproto.ExecCommandRequest) (*vmproto.ExecCommandResponse, error) {
	a.logger.Info("request to execute command", zap.String("command", in.Command), zap.Strings("args", in.Args))
	command := exec.Command(in.Command, in.Args...)
	output, err := command.Output()
	if err != nil {
		return nil, status.Errorf(codes.Internal, "command exited with error code: %v and output: %s", err, string(output))
	}
	return &vmproto.ExecCommandResponse{Output: output}, nil
}

// ActivateAdditionalNodes is the RPC call to activate additional nodes.
func (a *API) WriteFile(ctx context.Context, in *vmproto.WriteFileRequest) (*vmproto.WriteFileResponse, error) {
	a.logger.Info("request to write file", zap.String("path", in.Filepath), zap.String("name", in.Filename))
	if err := os.WriteFile(filepath.Join(in.Filepath, in.Filename), in.Content, os.ModeAppend); err != nil {
		a.logger.Error("failed to write file", zap.String("path", in.Filepath), zap.String("name", in.Filename), zap.Error(err))
		return nil, status.Errorf(codes.Internal, "file write failed exited with error code: %v", err)
	}
	return &vmproto.WriteFileResponse{}, nil
}
