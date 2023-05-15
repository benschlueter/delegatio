/* SPDX-License-Identifier: AGPL-3.0-only
 * Copyright (c) Edgeless Systems GmbH
 * Copyright (c) Benedict Schlueter
 * Copyright (c) Leonard Cohnen
 */

package main

import (
	"context"

	gradeapi "github.com/benschlueter/delegatio/grader/gradeAPI"
	"github.com/benschlueter/delegatio/internal/config"
	"go.uber.org/zap"
)

var version = "0.0.0"

func run(dialer gradeapi.Dialer, zapLoggerCore *zap.Logger) {
	defer func() { _ = zapLoggerCore.Sync() }()
	zapLoggerCore.Info("starting delegatio agent", zap.String("version", version), zap.String("commit", config.Commit))

	api := gradeapi.New(zapLoggerCore, dialer)
	points, err := api.SendGradingRequest(context.Background())
	if err != nil {
		zapLoggerCore.Fatal("failed to send grading request", zap.Error(err))
	}
	zapLoggerCore.Info("received points", zap.Int("points", points))
}
