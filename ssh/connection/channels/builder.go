/* SPDX-License-Identifier: AGPL-3.0-only
 * Copyright (c) Benedict Schlueter
 */

package channels

import (
	"context"
	"sync"

	"github.com/benschlueter/delegatio/ssh/connection/payload"
	"github.com/benschlueter/delegatio/ssh/local"
	"go.uber.org/zap"
	"golang.org/x/crypto/ssh"
)

// ChannelHandlerBuilder is a wrapper around an ssh.Channel and ssh.Requests.
type ChannelHandlerBuilder struct {
	channelType     string
	channel         ssh.Channel
	requests        <-chan *ssh.Request
	logger          *zap.Logger
	directTCPIPData *payload.ForwardTCPChannelOpen
	sharedData      *local.Shared

	onStartup    []func(context.Context, *callbackData)
	onRequest    []func(context.Context, *ssh.Request, *callbackData)
	onReqShell   []func(context.Context, *ssh.Request, *callbackData)
	onReqPty     []func(context.Context, *ssh.Request, *callbackData)
	onReqWinCh   []func(context.Context, *ssh.Request, *callbackData)
	onReqSubSys  []func(context.Context, *ssh.Request, *callbackData)
	onReqDefault []func(context.Context, *ssh.Request, *callbackData)
}

// NewChannelBuilder returns a new ChannelBuilder.
func NewChannelBuilder() *ChannelHandlerBuilder {
	return &ChannelHandlerBuilder{}
}

// WithChannelType sets the channel type.
func (b *ChannelHandlerBuilder) WithChannelType(channelType string) *ChannelHandlerBuilder {
	b.channelType = channelType
	return b
}

// SetChannel sets the channel.
func (b *ChannelHandlerBuilder) SetChannel(channel ssh.Channel) {
	b.channel = channel
}

// SetRequests sets the requests.
func (b *ChannelHandlerBuilder) SetRequests(requests <-chan *ssh.Request) {
	b.requests = requests
}

// SetOnStartup sets the onStartup callback.
func (b *ChannelHandlerBuilder) SetOnStartup(onStartup func(context.Context, *callbackData)) {
	b.onStartup = append(b.onStartup, onStartup)
}

// SetOnRequest sets the onRequest callback.
func (b *ChannelHandlerBuilder) SetOnRequest(onRequest func(context.Context, *ssh.Request, *callbackData)) {
	b.onRequest = append(b.onRequest, onRequest)
}

// SetOnReqShell sets the onReqShell callback.
func (b *ChannelHandlerBuilder) SetOnReqShell(onReqShell func(context.Context, *ssh.Request, *callbackData)) {
	b.onReqShell = append(b.onReqShell, onReqShell)
}

// SetOnReqPty sets the onReqPty callback.
func (b *ChannelHandlerBuilder) SetOnReqPty(onReqPty func(context.Context, *ssh.Request, *callbackData)) {
	b.onReqPty = append(b.onReqPty, onReqPty)
}

// SetOnReqWinCh sets the onReqWinCh callback.
func (b *ChannelHandlerBuilder) SetOnReqWinCh(onReqWinCh func(context.Context, *ssh.Request, *callbackData)) {
	b.onReqWinCh = append(b.onReqWinCh, onReqWinCh)
}

// SetOnReqSubSys sets the onReqSubSys callback.
func (b *ChannelHandlerBuilder) SetOnReqSubSys(onReqSubSys func(context.Context, *ssh.Request, *callbackData)) {
	b.onReqSubSys = append(b.onReqSubSys, onReqSubSys)
}

// SetOnReqDefault sets the onReqDefault callback.
func (b *ChannelHandlerBuilder) SetOnReqDefault(onReqDefault func(context.Context, *ssh.Request, *callbackData)) {
	b.onReqDefault = append(b.onReqDefault, onReqDefault)
}

// SetLog sets the logger.
func (b *ChannelHandlerBuilder) SetLog(logger *zap.Logger) {
	b.logger = logger
}

// SetSharedData sets the sharedData.
func (b *ChannelHandlerBuilder) SetSharedData(shared *local.Shared) {
	b.sharedData = shared
}

// SetDirectTCPIPData sets the directTCPIPData.
func (b *ChannelHandlerBuilder) SetDirectTCPIPData(directTCPIPData *payload.ForwardTCPChannelOpen) {
	b.directTCPIPData = directTCPIPData
}

// Build builds the channel.
func (b *ChannelHandlerBuilder) Build() (*channelHandler, error) {
	handler := &channelHandler{
		requests:       b.requests,
		serveCloseDone: make(chan struct{}),
		reqData: &callbackData{
			// TerminalSize handler is not closed at the moment (will be garbage collected anyways)
			terminalResizer: NewTerminalSizeHandler(10),
			log:             b.logger.Named("channel"),
			channel:         b.channel,
			wg:              &sync.WaitGroup{},
			directTCPIPData: b.directTCPIPData,
			Shared:          b.sharedData,
		},
		log:                b.logger.Named("channel"),
		onStartupCallback:  b.onStartup,
		onEveryReqCallback: b.onRequest,
		onDefaultCallback:  b.onReqDefault,
		funcMap: map[string][]func(context.Context, *ssh.Request, *callbackData){
			"shell":         b.onReqShell,
			"pty-req":       b.onReqPty,
			"window-change": b.onReqWinCh,
			"subsystem":     b.onReqSubSys,
		},
	}
	return handler, nil
}
