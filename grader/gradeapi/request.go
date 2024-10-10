/* SPDX-License-Identifier: AGPL-3.0-only
 * Copyright (c) Benedict Schlueter
 */

package gradeapi

import (
	"context"

	"github.com/benschlueter/delegatio/grader/gradeapi/gradeproto"
	"github.com/benschlueter/delegatio/grader/gradeapi/graders"
	"go.uber.org/zap"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/status"
)

func (a *API) updatePointsUser(_ context.Context, points int, user string) error {
	a.logger.Info("updating points for user", zap.String("user", user), zap.Int("points", points))
	// ToDO: Update points for user and add database connection / container
	return nil
}

// RequestGrading is the gRPC endpoint for requesting grading.
func (a *API) RequestGrading(ctx context.Context, in *gradeproto.RequestGradingRequest) (*gradeproto.RequestGradingResponse, error) {
	var points int
	var log []byte
	var err error
	/*
	 * How to authenticate the user?
	 * The user signs the request with a private key and the server verifies the signature
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
	toImplementName := "bschlueter"

	grader, err := graders.NewGraders(a.logger.Named(toImplementName), toImplementName)
	if err != nil {
		a.logger.Error("failed to create graders", zap.Error(err))
	}

	switch id := in.GetId(); id {
	case 1:
		a.logger.Info("received grading request for id 1")
		points, log, err = grader.GradeExerciseType1(ctx, in.GetSolution(), 1)
		if err != nil {
			return nil, status.Error(codes.InvalidArgument, "failed to grade exercise one")
		}
	case 2:
		a.logger.Info("received grading request for id 2")
	}

	if err := a.updatePointsUser(ctx, points, toImplementName); err != nil {
		return nil, status.Error(codes.Internal, "failed to update points")
	}

	return &gradeproto.RequestGradingResponse{Points: int32(points), Log: log}, nil
}

func (a *API) checkNonce(_ context.Context, _ string) error {
	return nil
}
