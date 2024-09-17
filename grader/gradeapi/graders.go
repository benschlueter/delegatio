/* SPDX-License-Identifier: AGPL-3.0-only
 * Copyright (c) Benedict Schlueter
 */

package gradeapi

import "context"

// Graders interface contains functions to access the state Graders data.
type Graders interface {
	GradeExerciseType1(ctx context.Context, solution []byte, id int) (int, []byte, error)
}
