package core

import (
	"go.uber.org/zap"
)

// Core is responsible for maintaining state information
// of the VM-agent. Currently we do not need any state.
type Core struct {
	zaplogger *zap.Logger
}

// NewCore creates and initializes a new Core object.
func NewCore(zapLogger *zap.Logger) (*Core, error) {
	c := &Core{
		zaplogger: zapLogger,
	}

	return c, nil
}
