/* SPDX-License-Identifier: AGPL-3.0-only
 * Copyright (c) Edgeless Systems GmbH
 */

// package osimage is used to handle osimages in the CI (uploading and maintenance).
package osimage

import (
	"io"
	"time"
)

// UploadRequest is a request to upload an os image.
type UploadRequest struct {
	Provider  string
	Version   string
	Variant   string
	Size      int64
	Timestamp time.Time
	Image     io.ReadSeeker
}
