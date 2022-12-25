package vmapi

import (
	"context"
	"net"
	"sync"

	"github.com/benschlueter/delegatio/core/vmapi/vmproto"
	"go.uber.org/zap"
)

// API is the API.
type API struct {
	mut    sync.Mutex
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

type Dialer interface {
	DialContext(ctx context.Context, network, address string) (net.Conn, error)
}
