/* SPDX-License-Identifier: AGPL-3.0-only
 * Copyright (c) Benedict Schlueter
 */

package vmapi

import (
	"context"

	"github.com/benschlueter/delegatio/agent/vmapi/vmproto"
)

func (a *API) getJoinDataKube(_ context.Context, in *vmproto.GetJoinDataKubeRequest) (*vmproto.GetJoinDataKubeResponse, error) {
	return nil, nil
}
