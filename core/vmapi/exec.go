package vmapi

import (
	"github.com/benschlueter/delegatio/core/vmapi/vmproto"
)

// ActivateAdditionalNodes is the RPC call to activate additional nodes.
func (a *API) ActivateAdditionalNodes(in *vmproto.ExecCommandRequest, srv vmproto.APIServer) (*vmproto.ExecCommandResponse, error) {
	return &vmproto.ExecCommandResponse{
		Output: "success",
	}, nil
}
