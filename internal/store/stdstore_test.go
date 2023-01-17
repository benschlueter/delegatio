package store

import "testing"

func TestStdStore(t *testing.T) {
	TestStore(t, func() (Store, error) {
		return NewStdStore(), nil
	})
}
