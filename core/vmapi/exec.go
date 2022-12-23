package vmapi

import (
	"context"
	"os/exec"

	"github.com/benschlueter/delegatio/core/vmapi/vmproto"
	"go.uber.org/zap"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/status"
)

// ActivateAdditionalNodes is the RPC call to activate additional nodes.
func (a *API) ExecCommand(ctx context.Context, in *vmproto.ExecCommandRequest) (*vmproto.ExecCommandResponse, error) {
	a.logger.Info("request to execute command", zap.String("command", in.Command), zap.Strings("args", in.Args))
	command := exec.Command(in.Command, in.Args...)
	output, err := command.CombinedOutput()
	if err != nil {
		return nil, status.Errorf(codes.Internal, "command exited with error code: %w and output %s", err, output)
	}
	return &vmproto.ExecCommandResponse{Output: output}, nil
}
