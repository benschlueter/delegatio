/* SPDX-License-Identifier: AGPL-3.0-only
 * Copyright (c) Edgeless Systems GmbH
 * Copyright (c) Benedict Schlueter
 * Copyright (c) Leonard Cohnen
 */

package main

import (
	"context"
	"os"

	gradeapi "github.com/benschlueter/delegatio/grader/gradeapi"
	"github.com/benschlueter/delegatio/internal/config"
	"go.uber.org/zap"
)

var version = "0.0.0"

func run(dialer gradeapi.Dialer, zapLoggerCore *zap.Logger) {
	defer func() { _ = zapLoggerCore.Sync() }()
	zapLoggerCore.Info("starting delegatio agent", zap.String("version", version), zap.String("commit", config.Commit))

	if len(os.Args) != 2 {
		zapLoggerCore.Fatal("usage: delegatio-agent <solution file>")
	}

	api, err := gradeapi.New(zapLoggerCore, dialer, nil)
	if err != nil {
		zapLoggerCore.Fatal("failed to create gradeapi", zap.Error(err))
	}
	points, err := api.SendGradingRequest(context.Background(), os.Args[1])
	if err != nil {
		zapLoggerCore.Fatal("failed to send grading request", zap.Error(err))
	}
	zapLoggerCore.Info("received points", zap.Int("points", points))
}
