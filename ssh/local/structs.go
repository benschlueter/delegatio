package local

import (
	"context"

	"github.com/benschlueter/delegatio/internal/config"
)

// Shared contains data shared between a ssh connection and the sub-channels opened by it.
type Shared struct {
	Namespace           string
	AuthenticatedUserID string
	ForwardFunc         func(context.Context, *config.KubeForwardConfig) error
	ExecFunc            func(context.Context, *config.KubeExecConfig) error
}
