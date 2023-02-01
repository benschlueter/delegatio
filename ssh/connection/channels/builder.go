/* SPDX-License-Identifier: AGPL-3.0-only
 * Copyright (c) Benedict Schlueter
 */

package channels

import (
	"context"
	"errors"
	"sync"

	"github.com/benschlueter/delegatio/ssh/connection/payload"
	"github.com/benschlueter/delegatio/ssh/local"
	"go.uber.org/zap"
	"golang.org/x/crypto/ssh"
)

// builder is a wrapper around an ssh.Channel and ssh.Requests.
type builder struct {
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

// NewBuilder returns a new ChannelBuilder.
func NewBuilder() *builder {
	return &builder{}
}

// WithChannelType sets the channel type.
func (b *builder) WithChannelType(channelType string) *builder {
	b.channelType = channelType
	return b
}

// SetChannel sets the channel.
func (b *builder) SetChannel(channel ssh.Channel) {
	b.channel = channel
}

// SetRequests sets the requests.
func (b *builder) SetRequests(requests <-chan *ssh.Request) {
	b.requests = requests
}

// SetOnStartup sets the onStartup callback.
func (b *builder) SetOnStartup(onStartup func(context.Context, *callbackData)) {
	b.onStartup = append(b.onStartup, onStartup)
}

// SetOnRequest sets the onRequest callback.
func (b *builder) SetOnRequest(onRequest func(context.Context, *ssh.Request, *callbackData)) {
	b.onRequest = append(b.onRequest, onRequest)
}

// SetOnReqShell sets the onReqShell callback.
func (b *builder) SetOnReqShell(onReqShell func(context.Context, *ssh.Request, *callbackData)) {
	b.onReqShell = append(b.onReqShell, onReqShell)
}

// SetOnReqPty sets the onReqPty callback.
func (b *builder) SetOnReqPty(onReqPty func(context.Context, *ssh.Request, *callbackData)) {
	b.onReqPty = append(b.onReqPty, onReqPty)
}

// SetOnReqWinCh sets the onReqWinCh callback.
func (b *builder) SetOnReqWinCh(onReqWinCh func(context.Context, *ssh.Request, *callbackData)) {
	b.onReqWinCh = append(b.onReqWinCh, onReqWinCh)
}

// SetOnReqSubSys sets the onReqSubSys callback.
func (b *builder) SetOnReqSubSys(onReqSubSys func(context.Context, *ssh.Request, *callbackData)) {
	b.onReqSubSys = append(b.onReqSubSys, onReqSubSys)
}

// SetOnReqDefault sets the onReqDefault callback.
func (b *builder) SetOnReqDefault(onReqDefault func(context.Context, *ssh.Request, *callbackData)) {
	b.onReqDefault = append(b.onReqDefault, onReqDefault)
}

// SetLog sets the logger.
func (b *builder) SetLog(logger *zap.Logger) {
	b.logger = logger
}

// SetSharedData sets the sharedData.
func (b *builder) SetSharedData(shared *local.Shared) {
	b.sharedData = shared
}

// SetDirectTCPIPData sets the directTCPIPData.
func (b *builder) SetDirectTCPIPData(directTCPIPData *payload.ForwardTCPChannelOpen) {
	b.directTCPIPData = directTCPIPData
}

// Build builds the channel.
func (b *builder) Build() (*channel, error) {
	if b.channel == nil {
		return nil, errors.New("channel is nil")
	}
	if b.requests == nil {
		return nil, errors.New("requests is nil")
	}
	if b.logger == nil {
		return nil, errors.New("logger is nil")
	}
	if b.sharedData == nil {
		return nil, errors.New("sharedData is nil")
	}
	if b.channelType == "direct-tcpip" && b.directTCPIPData == nil {
		return nil, errors.New("directTCPIPData is nil")
	}

	handler := &channel{
		requests:       b.requests,
		serveCloseDone: make(chan struct{}),
		reqData: &callbackData{
			// TerminalSize handler is not closed at the moment (will be garbage collected anyways)
			terminalResizer: NewTerminalSizeHandler(10),
			wg:              &sync.WaitGroup{},
			log:             b.logger.Named("channels").Named(b.channelType),
			channel:         b.channel,
			directTCPIPData: b.directTCPIPData,
			Shared:          b.sharedData,
		},
		log:                b.logger.Named(b.channelType),
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
