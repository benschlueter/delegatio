/* SPDX-License-Identifier: AGPL-3.0-only
 * Copyright (c) Benedict Schlueter
 */

package vmapi

import (
	"context"
	"errors"
	"io"
	"net"

	"github.com/benschlueter/delegatio/agent/vmapi/vmproto"
	"github.com/benschlueter/delegatio/internal/config"
	"go.uber.org/zap"
	"golang.org/x/sync/errgroup"
	"google.golang.org/grpc"
	"google.golang.org/grpc/credentials/insecure"
	"k8s.io/client-go/tools/remotecommand"
)

// VMAPI interface contains functions to access the agent.
type VMAPI interface {
	CreateExecInPodgRPC(context.Context, string, *config.KubeExecConfig) error
}

// API is the API.
type API struct {
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

func (a *API) dialInsecure(ctx context.Context, target string) (*grpc.ClientConn, error) {
	return grpc.DialContext(ctx, target,
		a.grpcWithDialer(),
		grpc.WithTransportCredentials(insecure.NewCredentials()),
		grpc.WithBlock(),
	)
}

func (a *API) grpcWithDialer() grpc.DialOption {
	return grpc.WithContextDialer(func(ctx context.Context, addr string) (net.Conn, error) {
		return a.dialer.DialContext(ctx, "tcp", addr)
	})
}

// Dialer is the dial interface. Necessary to stub network connections for local testing
// with bufconns.
type Dialer interface {
	DialContext(ctx context.Context, network, address string) (net.Conn, error)
}

// TODO: This code needs some refactoring / cleanup.

// CreateExecInPodgRPC creates a new exec in pod using gRPC connection to the endpoint agent.
func (a *API) CreateExecInPodgRPC(ctx context.Context, endpoint string, conf *config.KubeExecConfig) error {
	conn, err := a.dialInsecure(ctx, endpoint)
	if err != nil {
		return err
	}
	defer conn.Close()
	client := vmproto.NewAPIClient(conn)
	resp, err := client.ExecCommandStream(ctx)
	if err != nil {
		return err
	}
	err = resp.Send(&vmproto.ExecCommandStreamRequest{
		Content: &vmproto.ExecCommandStreamRequest_Command{
			Command: &vmproto.ExecCommandRequest{
				Command: conf.Command,
				Tty:     conf.Tty,
			},
		},
	})
	if err != nil {
		return err
	}

	g, ctx := errgroup.WithContext(ctx)
	g.Go(func() error {
		return a.receiver(ctx, resp, conf.Communication, conf.Communication)
	})
	g.Go(func() error {
		return a.sender(ctx, resp, conf.Communication)
	})
	g.Go(func() error {
		return a.termSizeHandler(ctx, resp, conf.WinQueue)
	})
	a.logger.Info("waiting for exec to finish")
	err = g.Wait()
	a.logger.Info("g wait returned")
	return err
}

func (a *API) termSizeHandler(ctx context.Context, resp vmproto.API_ExecCommandStreamClient, resizeData remotecommand.TerminalSizeQueue) error {
	queue := make(chan *remotecommand.TerminalSize, 1)
	go func() {
		for {
			data := resizeData.Next()
			queue <- data
			if data == nil {
				close(queue)
				return
			}
		}
	}()

	for {
		select {
		case <-ctx.Done():
			a.logger.Debug("terminalSizeHandler context done")
			return ctx.Err()
		case item := <-queue:
			if item == nil {
				a.logger.Debug("terminalSizeHandler queue closed")
				return errors.New("window size queue closed")
			}
			err := resp.Send(&vmproto.ExecCommandStreamRequest{
				Content: &vmproto.ExecCommandStreamRequest_Termsize{
					Termsize: &vmproto.TerminalSizeRequest{
						Width:  int32(item.Width),
						Height: int32(item.Height),
					},
				},
			})
			if err != nil {
				a.logger.Error("failed to send terminal size", zap.Error(err))
			}
			return err
		}
	}
}

func (a *API) receiver(ctx context.Context, resp vmproto.API_ExecCommandStreamClient, stdout io.Writer, stderr io.Writer) error {
	for {
		select {
		case <-ctx.Done():
			return ctx.Err()
		default:
			data, err := resp.Recv()
			a.logger.Info("after blocking call to resp.Recv")
			if err == io.EOF {
				return nil
			}
			if err != nil {
				a.logger.Error("failed to receive data from agent", zap.Error(err))
				return err
			}
			if len(data.GetStderr()) > 0 {
				stderr.Write(data.GetStderr())
			}
			if len(data.GetStdout()) > 0 {
				stdout.Write(data.GetStdout())
			}
		}
	}
}

// sender should not signal error if we fail to send something to the server.
// The reader will pull the remaining data from the server and will return the errorcode.
func (a *API) sender(ctx context.Context, resp vmproto.API_ExecCommandStreamClient, stdin io.Reader) error {
	// g, _ := errgroup.WithContext(ctx)
	errChan := make(chan error, 1)

	// TODO: try to kill this goroutine when the context is done / synchronously
	// Don't wait for the garbage collector to clean up this goroutine.
	go func() {
		copier := make([]byte, 4096)
		for {
			n, err := stdin.Read(copier)
			a.logger.Info("after blocking call to stdin Read")
			if err == io.EOF {
				a.logger.Info("received EOF from stdin")
				errChan <- err
				return
			}
			if err != nil {
				a.logger.Error("failed to receive data from ssh connection", zap.Error(err))
				errChan <- err
				return
			}
			err = resp.Send(&vmproto.ExecCommandStreamRequest{
				Content: &vmproto.ExecCommandStreamRequest_Stdin{
					Stdin: copier[:n],
				},
			})
			if err != nil {
				a.logger.Error("failed to send data to agent", zap.Error(err))
				errChan <- err
				return
			}
		}
	}()
	for {
		select {
		case <-ctx.Done():
			return nil
		case <-errChan:
			close(errChan)
			return nil
		}
	}
}
