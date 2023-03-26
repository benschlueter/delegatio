/* SPDX-License-Identifier: AGPL-3.0-only
 * Copyright (c) Benedict Schlueter
 */

package vmapi

import (
	"errors"
	"sync"
)

// TerminalSize is the struct holding the size data.
type TerminalSize struct {
	Width  uint16
	Height uint16
}

// TerminalSizeHandler stores the Height and Width of a terminal.
type TerminalSizeHandler struct {
	queue    chan *TerminalSize
	closed   bool
	capacity int
	mux      sync.Mutex
}

// NewTerminalSizeHandler creates a new Winsize.
func NewTerminalSizeHandler(cap int) *TerminalSizeHandler {
	return &TerminalSizeHandler{
		queue:    make(chan *TerminalSize, cap),
		capacity: cap,
	}
}

// Next returns the size. The chanel must be served. Otherwise the connection will hang.
func (w *TerminalSizeHandler) Next() *TerminalSize {
	return <-w.queue
}

// Fill appends the data to the queue.
func (w *TerminalSizeHandler) Fill(data *TerminalSize) error {
	w.mux.Lock()
	defer w.mux.Unlock()
	if w.closed {
		return ErrQueueClosed
	}
	if len(w.queue) == w.capacity {
		return ErrQueueFull
	}
	w.queue <- data
	return nil
}

// Close closes the winsize queue and chan.
func (w *TerminalSizeHandler) Close() {
	w.mux.Lock()
	defer w.mux.Unlock()
	if !w.closed {
		w.closed = true
		close(w.queue)
		return
	}
}

var (
	// ErrQueueFull is returned when the queue is full.
	ErrQueueFull = errors.New("winsize: queue is full")
	// ErrQueueClosed is returned when a channel is full.
	ErrQueueClosed = errors.New("chan is closed, cannot append data")
)
