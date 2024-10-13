/* SPDX-License-Identifier: AGPL-3.0-only
 * Copyright (c) Edgeless Systems GmbH
 * Copyright (c) Benedict Schlueter
 * Copyright (c) Leonard Cohnen
 */

package main

import (
	"context"
	"crypto"
	"crypto/rand"
	"crypto/rsa"
	"crypto/sha512"
	"os"

	gradeapi "github.com/benschlueter/delegatio/grader/gradeapi"
	"github.com/benschlueter/delegatio/internal/config"
	"go.uber.org/zap"
	"golang.org/x/crypto/ssh"
)

var version = "0.0.0"

// Not really clean, giving the gradeapi nil here (should be done differently)
func run(dialer gradeapi.Dialer, zapLoggerCore *zap.Logger) {
	defer func() { _ = zapLoggerCore.Sync() }()
	zapLoggerCore.Info("starting delegatio agent", zap.String("version", version), zap.String("commit", config.Commit))

	if len(os.Args) != 2 {
		zapLoggerCore.Fatal("usage: delegatio-agent <solution file>")
	}
	solution, err := os.ReadFile(os.Args[1])
	if err != nil {
		zapLoggerCore.Fatal("reading solution file", zap.Error(err))
	}
	privKeyData, err := os.ReadFile("/root/.ssh/delegatio_priv_key")
	if err != nil {
		zapLoggerCore.Fatal("opening private key file ", zap.Error(err))
	}
	signKeyIface, err := ssh.ParseRawPrivateKey(privKeyData)
	if err != nil {
		zapLoggerCore.Fatal("parsing private key", zap.Error(err))
	}
	signKey, ok := signKeyIface.(*rsa.PrivateKey)
	if !ok {
		zapLoggerCore.Fatal("private key is not rsa")
	}
	hashSolution := sha512.Sum512(solution)
	signature, err := rsa.SignPKCS1v15(rand.Reader, signKey, crypto.SHA512, hashSolution[:])
	if err != nil {
		zapLoggerCore.Fatal("signing solution", zap.Error(err))
	}

	api, err := gradeapi.New(zapLoggerCore, dialer, false)
	if err != nil {
		zapLoggerCore.Fatal("create gradeapi", zap.Error(err))
	}
	points, err := api.SendGradingRequest(
		context.Background(),
		solution,
		signature,
		os.Getenv(config.UUIDEnvVariable))
	if err != nil {
		zapLoggerCore.Fatal("send grading request", zap.Error(err))
	}
	zapLoggerCore.Info("received points", zap.Int("points", points))
}
