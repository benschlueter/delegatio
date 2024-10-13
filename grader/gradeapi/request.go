/* SPDX-License-Identifier: AGPL-3.0-only
 * Copyright (c) Benedict Schlueter
 */

package gradeapi

import (
	"context"
	"crypto"
	"crypto/rsa"
	"crypto/sha512"

	"github.com/benschlueter/delegatio/grader/gradeapi/gradeproto"
	"github.com/benschlueter/delegatio/grader/gradeapi/graders"
	"github.com/benschlueter/delegatio/internal/config"
	"go.uber.org/zap"
	"golang.org/x/crypto/ssh"
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
	uuid := in.GetStudentId()
	/*
	 * How to authenticate the user?
	 * The user signs the request with a private key and the server verifies the signature
	 */
	/* 	p, _ := peer.FromContext(ctx)
	   	requestEndpoint := p.Addr.String() */
	// ToDO: use a unique ID from the USER
	a.logger.Info("received grading request; verifying identity")
	if err := a.checkSignature(ctx, uuid, in.GetSignature(), in.GetSolution()); err != nil {
		return nil, err
	}

	grader, err := graders.NewGraders(a.logger.Named(uuid), uuid)
	if err != nil {
		a.logger.Error("failed to create graders", zap.Error(err))
	}

	switch id := in.GetExerciseId(); id {
	case 1:
		a.logger.Info("received grading request for id 1")
		points, log, err = grader.GradeExerciseType1(ctx, in.GetSolution(), 1)
		if err != nil {
			return nil, status.Error(codes.InvalidArgument, "failed to grade exercise one")
		}
	case 2:
		a.logger.Info("received grading request for id 2")
	}

	if err := a.updatePointsUser(ctx, points, uuid); err != nil {
		return nil, status.Error(codes.Internal, "failed to update points")
	}

	return &gradeproto.RequestGradingResponse{Points: int32(points), Log: log}, nil
}

func (a *API) checkSignature(_ context.Context, UUID string, signature, solution []byte) error {
	a.logger.Info("checking signature", zap.String("studentID", UUID))
	exists, err := a.data().UUIDExists(UUID)
	if err != nil {
		return err
	}
	if !exists {
		return status.Error(codes.NotFound, "user not found")
	}
	var userData config.UserInformation
	if err := a.data().GetUUIDData(UUID, &userData); err != nil {
		return status.Error(codes.FailedPrecondition, "failed to get user data")
	}
	a.logger.Info("got user data", zap.String("publicKey", string(userData.PubKey)))
	sshPubKey, err := ssh.ParsePublicKey(userData.PubKey)
	if err != nil {
		return status.Error(codes.Internal, "failed to unmarshal public key")
	}
	pubKeyNewIface := sshPubKey.(ssh.CryptoPublicKey)
	pubKewNewIfaceTwo := pubKeyNewIface.CryptoPublicKey()
	rsaPubKey := pubKewNewIfaceTwo.(*rsa.PublicKey)
	hashSolution := sha512.Sum512(solution)
	if err := rsa.VerifyPKCS1v15(rsaPubKey, crypto.SHA512, hashSolution[:], signature); err != nil {
		return status.Error(codes.Unauthenticated, "signature check")
	}
	return nil
}
