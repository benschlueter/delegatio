package core

import (
	"sync"
	"time"

	"go.uber.org/zap"
)

type Core struct {
	mut            sync.Mutex
	zaplogger      *zap.Logger
	lastHeartbeats map[string]time.Time
	cancelFuncs    []func()
}

// NewCore creates and initializes a new Core object.
func NewCore(zapLogger *zap.Logger) (*Core, error) {
	c := &Core{
		zaplogger: zapLogger,
	}

	return c, nil
}
