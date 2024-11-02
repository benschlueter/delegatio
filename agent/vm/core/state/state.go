/* SPDX-License-Identifier: AGPL-3.0-only
 * Copyright (c) Benedict Schlueter
 */

package state

import (
	"fmt"
	"sync/atomic"
)

//go:generate stringer -type=State

// State is a peer's state.
//
// State's methods are thread safe. Get the State's value using the Get method if
// you're accessing it concurrently. Otherwise, you may access it directly.
type State uint32

const (
	// Uninitialized is the initial state.
	Uninitialized State = iota
	// AcceptingInit is the state where the peer is accepting an init message.
	AcceptingInit
	// JoiningCluster is the state where the peer is about to join the cluster.
	JoiningCluster
	// Initialized is the state where the peer is initializing.
	Initialized
	// Failed is the state where the peer has failed.
	Failed
	maxState
)

// State's methods should be thread safe. As we only need to protect
// one primitive value, we can use atomic operations.

// Get gets the state in a thread-safe manner.
func (s *State) Get() State {
	return State(atomic.LoadUint32((*uint32)(s)))
}

// Require checks if the state is one of the desired ones and returns an error otherwise.
func (s *State) Require(states ...State) error {
	this := s.Get()
	for _, st := range states {
		if st == this {
			return nil
		}
	}
	return fmt.Errorf("server is not in expected state: require one of %v, but this is %v", states, this)
}

// Advance advances the state.
func (s *State) Advance(newState State) {
	curState := State(atomic.SwapUint32((*uint32)(s), uint32(newState)))
	if !(curState < newState && newState < maxState) {
		panic(fmt.Errorf("cannot advance from %v to %v", curState, newState))
	}
}
