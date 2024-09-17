/* SPDX-License-Identifier: AGPL-3.0-only
 * Copyright (c) Benedict Schlueter
 */

package gradeapi

import (
	"context"

	"github.com/benschlueter/delegatio/grader/gradeapi/gradeproto"
	"go.uber.org/zap"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/status"
)

func (a *API) updatePointsUser(ctx context.Context, points int, user string) error {
	a.logger.Info("updating points for user", zap.String("user", user), zap.Int("points", points))

	return nil
}

// RequestGrading is the gRPC endpoint for requesting grading.
func (a *API) RequestGrading(ctx context.Context, in *gradeproto.RequestGradingRequest) (*gradeproto.RequestGradingResponse, error) {
	var points int
	var log []byte
	var err error
	/*
	 * How to authenticate the user?
	 * Either the user signs the request with a private key and the server verifies the signature
	 * Or the server does a reverse RPC and reads the contents of the nonce file / checks the existence
	 */
	/* 	p, _ := peer.FromContext(ctx)
	   	requestEndpoint := p.Addr.String() */
	// ToDO: use a unique ID from the USER
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
		points, log, err = a.grader.GradeExerciseType1(ctx, in.GetSolution(), 1)
		if err != nil {
			return nil, status.Error(codes.InvalidArgument, "failed to grade exercise one")
		}
	case 2:
		a.logger.Info("received grading request for id 2")
	}

	return &gradeproto.RequestGradingResponse{Points: int32(points), Log: log}, nil
}

func (a *API) checkNonce(_ context.Context, _ string) error {
	return nil
}
