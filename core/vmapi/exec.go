package vmapi

import (
	"bytes"
	"io"
	"os/exec"

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

// ActivateAdditionalNodes is the RPC call to activate additional nodes.
func (a *API) ExecCommand(in *vmproto.ExecCommandRequest, srv vmproto.API_ExecCommandServer) error {
	a.logger.Info("request to execute command", zap.String("command", in.Command), zap.Strings("args", in.Args))
	command := exec.Command(in.Command, in.Args...)
	streamer := streamWriter{forward: func(b []byte) error {
		return srv.Send(&vmproto.ExecCommandResponse{
			Content: &vmproto.ExecCommandResponse_Log{
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
		return status.Errorf(codes.Internal, "command exited with error code: %v and output %s", err, "output")
	}

	command.Wait()
	return srv.Send(&vmproto.ExecCommandResponse{Content: &vmproto.ExecCommandResponse_Output{Output: stdoutBuf.Bytes()}})
}
