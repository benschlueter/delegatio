/* SPDX-License-Identifier: AGPL-3.0-only
 * Copyright (c) Benedict Schlueter
 */

package graders

import "context"

// Exercises interface contains functions to access the state Exercises data.
type Exercises interface {
	GetFiles(ctx context.Context) ([]byte, error)
}
