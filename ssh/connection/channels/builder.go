/* SPDX-License-Identifier: AGPL-3.0-only
 * Copyright (c) Benedict Schlueter
 */

package channels

import (
	"context"
	"errors"
	"sync"

	"github.com/benschlueter/delegatio/ssh/connection/payload"
	"github.com/benschlueter/delegatio/ssh/kubernetes"
	"go.uber.org/zap"
	"golang.org/x/crypto/ssh"
)

// Builder is builder for channels.
type Builder struct {
	channelType     string
	channel         ssh.Channel
	requests        <-chan *ssh.Request
	logger          *zap.Logger
	directTCPIPData *payload.ForwardTCPChannelOpen
	k8sAPIUser      kubernetes.K8sAPIUser

	onStartup    []func(context.Context, *callbackData)
	onRequest    []func(context.Context, *ssh.Request, *callbackData)
	onReqShell   []func(context.Context, *ssh.Request, *callbackData)
	onReqPty     []func(context.Context, *ssh.Request, *callbackData)
	onReqWinCh   []func(context.Context, *ssh.Request, *callbackData)
	onReqSubSys  []func(context.Context, *ssh.Request, *callbackData)
	onReqDefault []func(context.Context, *ssh.Request, *callbackData)
}

// NewBuilder returns a new ChannelBuilder.
func NewBuilder() *Builder {
	return &Builder{}
}

// WithChannelType sets the channel type.
func (b *Builder) WithChannelType(channelType string) *Builder {
	b.channelType = channelType
	return b
}

// SetChannel sets the channel.
func (b *Builder) SetChannel(channel ssh.Channel) {
	b.channel = channel
}

// SetRequests sets the requests.
func (b *Builder) SetRequests(requests <-chan *ssh.Request) {
	b.requests = requests
}

// SetOnStartup sets the onStartup callback.
func (b *Builder) SetOnStartup(onStartup func(context.Context, *callbackData)) {
	b.onStartup = append(b.onStartup, onStartup)
}

// SetOnRequest sets the onRequest callback.
func (b *Builder) SetOnRequest(onRequest func(context.Context, *ssh.Request, *callbackData)) {
	b.onRequest = append(b.onRequest, onRequest)
}

// SetOnReqShell sets the onReqShell callback.
func (b *Builder) SetOnReqShell(onReqShell func(context.Context, *ssh.Request, *callbackData)) {
	b.onReqShell = append(b.onReqShell, onReqShell)
}

// SetOnReqPty sets the onReqPty callback.
func (b *Builder) SetOnReqPty(onReqPty func(context.Context, *ssh.Request, *callbackData)) {
	b.onReqPty = append(b.onReqPty, onReqPty)
}

// SetOnReqWinCh sets the onReqWinCh callback.
func (b *Builder) SetOnReqWinCh(onReqWinCh func(context.Context, *ssh.Request, *callbackData)) {
	b.onReqWinCh = append(b.onReqWinCh, onReqWinCh)
}

// SetOnReqSubSys sets the onReqSubSys callback.
func (b *Builder) SetOnReqSubSys(onReqSubSys func(context.Context, *ssh.Request, *callbackData)) {
	b.onReqSubSys = append(b.onReqSubSys, onReqSubSys)
}

// SetOnReqDefault sets the onReqDefault callback.
func (b *Builder) SetOnReqDefault(onReqDefault func(context.Context, *ssh.Request, *callbackData)) {
	b.onReqDefault = append(b.onReqDefault, onReqDefault)
}

// SetLog sets the logger.
func (b *Builder) SetLog(logger *zap.Logger) {
	b.logger = logger
}

// SetK8sUserAPI sets the k8sAPIUser.
func (b *Builder) SetK8sUserAPI(api kubernetes.K8sAPIUser) {
	b.k8sAPIUser = api
}

// SetDirectTCPIPData sets the directTCPIPData.
func (b *Builder) SetDirectTCPIPData(directTCPIPData *payload.ForwardTCPChannelOpen) {
	b.directTCPIPData = directTCPIPData
}

// Build builds the channel.
func (b *Builder) Build() (*Handler, error) {
	if b.channel == nil {
		return nil, errors.New("channel is nil")
	}
	if b.requests == nil {
		return nil, errors.New("requests is nil")
	}
	if b.logger == nil {
		return nil, errors.New("logger is nil")
	}
	if b.k8sAPIUser == nil {
		return nil, errors.New("k8sAPIUser is nil")
	}
	if b.channelType == "direct-tcpip" && b.directTCPIPData == nil {
		return nil, errors.New("directTCPIPData is nil")
	}

	handler := &Handler{
		requests:       b.requests,
		serveCloseDone: make(chan struct{}),
		reqData: &callbackData{
			// TerminalSize handler is not closed at the moment (will be garbage collected anyways)
			terminalResizer: NewTerminalSizeHandler(10),
			wg:              &sync.WaitGroup{},
			log:             b.logger.Named("channels").Named(b.channelType),
			channel:         b.channel,
			directTCPIPData: b.directTCPIPData,
			K8sAPIUser:      b.k8sAPIUser,
		},
		log:               b.logger.Named(b.channelType),
		onStartupCallback: b.onStartup,
		onRequestCallback: b.onRequest,
		onDefaultCallback: b.onReqDefault,
		funcMap: map[string][]func(context.Context, *ssh.Request, *callbackData){
			"shell":         b.onReqShell,
			"pty-req":       b.onReqPty,
			"window-change": b.onReqWinCh,
			"subsystem":     b.onReqSubSys,
		},
	}
	return handler, nil
}
