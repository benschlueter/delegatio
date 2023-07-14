/* SPDX-License-Identifier: AGPL-3.0-only
 * Copyright (c) Benedict Schlueter
 */

package vmapi

import (
	"bytes"
	"context"
	"io"
	"os/exec"
	"time"

	"github.com/benschlueter/delegatio/agent/vmapi/vmproto"
	"go.uber.org/zap"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/status"
)

// GetJoinDataKube returns the join data for the kubernetes cluster. This function will be used by all master nodes except the first one and all worker nodes.
func (a *API) GetJoinDataKube(_ context.Context, _ *vmproto.GetJoinDataKubeRequest) (*vmproto.GetJoinDataKubeResponse, error) {
	if !a.core.IsInReadyState() {
		return nil, status.Errorf(codes.FailedPrecondition, "cluster is not ready")
	}
	token, err := a.core.GetJoinToken(5 * time.Minute)
	if err != nil {
		return nil, status.Errorf(codes.Internal, "get join token %v", err)
	}
	files, err := a.core.GetControlPlaneCertificatesAndKeys()
	if err != nil {
		return nil, status.Errorf(codes.Internal, "get certs %v", err)
	}

	var controlPlaneFiles []*vmproto.File
	for k, v := range files {
		controlPlaneFiles = append(controlPlaneFiles, &vmproto.File{
			Name:    k,
			Content: v,
		})
	}

	return &vmproto.GetJoinDataKubeResponse{
		JoinToken: &vmproto.JoinToken{
			Token:             token.Token,
			CaCertHash:        token.CACertHashes[0],
			ApiServerEndpoint: token.APIServerEndpoint,
		},
		Files: controlPlaneFiles,
	}, nil
}

// InitFirstMaster executes the kubeadm init command on the first master node. The subsequent master nodes will join the cluster automatically.
func (a *API) InitFirstMaster(in *vmproto.InitFirstMasterRequest, srv vmproto.API_InitFirstMasterServer) error {
	a.logger.Info("request to execute command", zap.String("command", in.Command), zap.Strings("args", in.Args))
	a.core.SetJoiningCluster()
	command := exec.Command(in.Command, in.Args...)
	streamer := &streamWriterWrapper{forwardFunc: func(b []byte) error {
		return srv.Send(&vmproto.InitFirstMasterResponse{
			Content: &vmproto.InitFirstMasterResponse_Log{
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

	// So the core has access to the kubernetes client.
	if err := a.core.ConnectToKubernetes(); err != nil {
		return status.Errorf(codes.Internal, "failed to connect to kubernetes: %v", err)
	}
	a.core.SetInitialized()
	return srv.Send(&vmproto.InitFirstMasterResponse{Content: &vmproto.InitFirstMasterResponse_Output{Output: stdoutBuf.Bytes()}})
}
