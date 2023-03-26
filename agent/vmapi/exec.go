/* SPDX-License-Identifier: AGPL-3.0-only
 * Copyright (c) Benedict Schlueter
 */

package vmapi

import (
	"bytes"
	"context"
	"fmt"
	"io"
	"os/exec"

	"github.com/benschlueter/delegatio/agent/vmapi/vmproto"
	"github.com/creack/pty"
	"github.com/sirupsen/logrus"
	"go.uber.org/zap"
	"golang.org/x/sys/unix"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/status"
)

func setSize(fd uintptr, size *TerminalSize) error {
	winsize := &unix.Winsize{Row: size.Height, Col: size.Width}
	return unix.IoctlSetWinsize(int(fd), unix.TIOCSWINSZ, winsize)
}

func (a *API) ttyCmd(execCmd *exec.Cmd, stdin io.Reader, stdout io.WriteCloser, handler *TerminalSizeHandler) error {
	p, err := pty.Start(execCmd)
	if err != nil {
		return err
	}
	defer p.Close()

	// make sure to close the stdout stream
	defer stdout.Close()

	go func() {
		for {
			err := setSize(p.Fd(), handler.Next())
			if err != nil {
				a.logger.Error("setting size", zap.Error(err))
			}
		}
	}()

	var stdinErr, stdoutErr error
	if stdin != nil {
		go func() { _, stdinErr = io.Copy(p, stdin) }()
	}

	if stdout != nil {
		go func() { _, stdoutErr = io.Copy(stdout, p) }()
	}

	err = execCmd.Wait()

	if stdinErr != nil {
		logrus.Warnf("Stdin copy error: %v", stdinErr)
	}
	if stdoutErr != nil {
		logrus.Warnf("Stdout copy error: %v", stdoutErr)
	}

	return err
}

// ExecCommandStream executes a command in the VM and streams the output to the caller.
// This is useful if the command needs much time to run and we want to log the current state, i.e. kubeadm.
func (a *API) ExecCommandStream(srv vmproto.API_ExecCommandStreamServer) error {
	a.logger.Info("ExecCommandStream")
	in, err := srv.Recv()
	if err != nil {
		a.logger.Error("error receiving command", zap.Error(err))
		return status.Error(codes.InvalidArgument, "error receiving input")
	}
	a.logger.Info("Received client request")
	command := in.GetCommand()
	if command == nil {
		a.logger.Error("no command received")
		return status.Error(codes.InvalidArgument, "no command received")
	}
	execCommand := exec.Command(command.Command, command.Args...)

	errorStreamWriter := &streamWriterWrapper{
		forwardFunc: func(b []byte) error {
			return srv.Send(&vmproto.ExecCommandStreamResponse{
				Content: &vmproto.ExecCommandStreamResponse_Stderr{
					Stderr: b,
				},
			})
		},
	}
	stdoutStreamWrtier := &streamWriterWrapper{
		forwardFunc: func(b []byte) error {
			return srv.Send(&vmproto.ExecCommandStreamResponse{
				Content: &vmproto.ExecCommandStreamResponse_Stdout{
					Stdout: b,
				},
			})
		},
	}

	reader, writer := io.Pipe()

	sizeHandler := NewTerminalSizeHandler(10)
	go func() {
		for {
			in, err := srv.Recv()
			if err != nil {
				a.logger.Error("error receiving stdin", zap.Error(err))
				writer.CloseWithError(err)
				return
			}
			if input := in.GetStdin(); input != nil {
				_, err = writer.Write(input)
				if err != nil {
					a.logger.Error("error writing stdin", zap.Error(err))
					writer.CloseWithError(err)
					return
				}
			}
			if resize := in.GetTermsize(); resize != nil {
				err := sizeHandler.Fill(&TerminalSize{
					Width:  uint16(resize.Width),
					Height: uint16(resize.Height),
				})
				if err != nil {
					a.logger.Error("adding resize request", zap.Error(err))
				}
			}
		}
	}()

	var cmdErr error
	if command.Tty {
		cmdErr = a.ttyCmd(execCommand, reader, stdoutStreamWrtier, sizeHandler)
	} else {
		execCommand.Stdout = stdoutStreamWrtier
		execCommand.Stderr = errorStreamWriter
		execCommand.Stdin = reader

		a.logger.Info("starting command", zap.String("command", command.Command), zap.Strings("args", command.Args))
		if err := execCommand.Start(); err != nil {
			a.logger.Error("command start exited with error", zap.Error(err))
			return status.Errorf(codes.Internal, "command exited with error code: %v", err)
		}

		if err := execCommand.Wait(); err != nil {
			a.logger.Error("command wait exited with error", zap.Error(err))
			return status.Errorf(codes.Internal, "command exited with error code: %v", err)
		}
	}

	return cmdErr
}

// ExecCommandReturnStream executes a command in the VM and streams the output to the caller.
// This is useful if the command needs much time to run and we want to log the current state, i.e. kubeadm.
func (a *API) ExecCommandReturnStream(in *vmproto.ExecCommandRequest, srv vmproto.API_ExecCommandReturnStreamServer) error {
	a.logger.Info("request to execute command", zap.String("command", in.Command), zap.Strings("args", in.Args))
	command := exec.Command(in.Command, in.Args...)
	streamer := &streamWriterWrapper{forwardFunc: func(b []byte) error {
		return srv.Send(&vmproto.ExecCommandReturnStreamResponse{
			Content: &vmproto.ExecCommandReturnStreamResponse_Log{
				Log: &vmproto.Log{
					Message: string(b),
				},
			},
		})
	}}
	var stdoutBuf, stderrBuf bytes.Buffer

	command.Stdout = io.MultiWriter(streamer, &stdoutBuf)
	command.Stderr = io.MultiWriter(streamer, &stderrBuf)

	if err := command.Start(); err != nil {
		return status.Errorf(codes.Internal, "command exited with error code: %v and output: %s", err, stdoutBuf.Bytes())
	}

	if err := command.Wait(); err != nil {
		return status.Errorf(codes.Internal, "command exited with error code: %v and output: %s", err, stdoutBuf.Bytes())
	}
	return srv.Send(&vmproto.ExecCommandReturnStreamResponse{Content: &vmproto.ExecCommandReturnStreamResponse_Output{Output: stdoutBuf.Bytes()}})
}

// ExecCommand executes a command in the VM.
func (a *API) ExecCommand(_ context.Context, in *vmproto.ExecCommandRequest) (*vmproto.ExecCommandResponse, error) {
	a.logger.Info("request to execute command", zap.String("command", in.Command), zap.Strings("args", in.Args))
	command := exec.Command(in.Command, in.Args...)
	output, err := command.Output()
	if err != nil {
		return nil, status.Errorf(codes.Internal, "command exited with error code: %v and output: %s", err, string(output))
	}
	return &vmproto.ExecCommandResponse{Output: output}, nil
}

type streamWriterWrapper struct {
	forwardFunc func([]byte) error
}

func (sw *streamWriterWrapper) Write(p []byte) (int, error) {
	fmt.Println("write")
	if err := sw.forwardFunc(p); err != nil {
		return 0, err
	}
	return len(p), nil
}

func (sw *streamWriterWrapper) Close() error {
	return nil
}
