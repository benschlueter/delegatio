/* SPDX-License-Identifier: AGPL-3.0-only
 * Copyright (c) Benedict Schlueter
 */

package gradeapi

import (
	"context"

	"github.com/benschlueter/delegatio/grader/gradeAPI/gradeproto"
	"go.uber.org/zap"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/status"
)

// RequestGrading is the gRPC endpoint for requesting grading.
func (a *API) RequestGrading(ctx context.Context, in *gradeproto.RequestGradingRequest) (*gradeproto.RequestGradingResponse, error) {
	var points int
	/* 	p, _ := peer.FromContext(ctx)
	   	requestEndpoint := p.Addr.String() */

	if in.GetSubmit() {
		a.logger.Info("received grading request; verifying identity")
		a.logger.Debug("nonce check passed", zap.String("nonce", in.GetNonce()))
		// TODO: Verify identity
		if err := a.checkNonce(ctx, in.GetNonce()); err != nil {
			return nil, status.Error(codes.Unauthenticated, "nonce check failed")
		}
	}

	switch id := in.GetId(); id {
	case 1:
		a.logger.Info("received grading request for id 1")
		points = 100
	case 2:
		a.logger.Info("received grading request for id 2")
	}

	return &gradeproto.RequestGradingResponse{Points: int32(points)}, nil
}

func (a *API) checkNonce(_ context.Context, _ string) error {
	return nil
}
