/* SPDX-License-Identifier: AGPL-3.0-only
 * Copyright (c) Benedict Schlueter
 */

package graders

import (
	"context"
	"fmt"
	"os"
	"path/filepath"
	"strings"
	"time"

	"go.uber.org/zap"
)

/*
 * Type 1 exercises take a program from the user and execute it on given input files.
 * The input files are stored in exercises/exerciseX/
 */
// TODO: Put input files in exercises/exerciseX/input and have a json/yaml file with the expected output

func (g *Graders) GradeExerciseType1(ctx context.Context, solution []byte, id int) (int, []byte, error) {
	g.logger.Info("grading exercise type 1", zap.Int("id", id))
	defer g.logger.Info("finished grading exercise type 1", zap.Int("id", id))
	file, err := g.writeFileToDisk(ctx, solution)
	if err != nil {
		return 0, nil, err
	}
	defer func() {
		file.Close()
	}()
	// TODO: Use Namespaces to run the code in a sandboxed environment

	inputDir := filepath.Join("/exercises/", fmt.Sprintf("exercise%d", id))
	files, err := os.ReadDir(inputDir)
	if err != nil {
		return 0, nil, err
	}
	ctx, cancel := context.WithDeadline(ctx, time.Now().Add(g.totalExecTimeout))
	defer cancel()
	for _, f := range files {
		if !f.IsDir() {
			inputFilePath := filepath.Join(inputDir, f.Name())
			output, err := g.executeCommand(ctx, file.Name(), inputFilePath)
			if err != nil {
				g.logger.Error("failed to execute command", zap.String("command", file.Name()), zap.String("arg", inputFilePath), zap.Error(err), zap.Error(ctx.Err()))
				return 0, nil, err
			}
			if !strings.Contains(string(output), f.Name()[:len(f.Name())-4]) {
				g.logger.Info("output does not match expected output", zap.String("output", string(output)), zap.String("expected", f.Name()[:len(f.Name())-4]))
				return 0, output, nil
			}
		}
	}

	return 100, nil, nil
}
