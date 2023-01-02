package vmapi

import (
	"context"
	"net"

	"github.com/benschlueter/delegatio/client/vmapi/vmproto"
	"go.uber.org/zap"
)

// API is the API.
type API struct {
	logger *zap.Logger
	core   Core
	dialer Dialer
	vmproto.UnimplementedAPIServer
}

// New creates a new API.
func New(logger *zap.Logger, core Core, dialer Dialer) *API {
	return &API{
		logger: logger,
		core:   core,
		dialer: dialer,
	}
}

// Dialer is the dial interface. Necessary to stub network connections for local testing
// with bufconns.
type Dialer interface {
	DialContext(ctx context.Context, network, address string) (net.Conn, error)
}
