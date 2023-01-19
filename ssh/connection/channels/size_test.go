/* SPDX-License-Identifier: AGPL-3.0-only
 * Copyright (c) Benedict Schlueter
 */

package channels

import (
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"go.uber.org/goleak"
	"k8s.io/client-go/tools/remotecommand"
)

func TestFill(t *testing.T) {
	defer goleak.VerifyNone(t)
	testCases := map[string]struct {
		qSize        int
		firstObjCnt  int
		secondObjCnt int
		expectedErr  error
		closed       bool
	}{
		"one request": {
			qSize:        1,
			secondObjCnt: 1,
		},
		"consumer too slow": {
			qSize:        1,
			firstObjCnt:  1,
			secondObjCnt: 2,
			expectedErr:  ErrQueueFull,
		},
		"queue already closed": {
			qSize:        1,
			firstObjCnt:  1,
			closed:       true,
			secondObjCnt: 2,
			expectedErr:  ErrQueueClosed,
		},
	}

	for name, tc := range testCases {
		t.Run(name, func(t *testing.T) {
			assert := assert.New(t)
			require := require.New(t)
			wsize := NewTerminalSizeHandler(tc.qSize)
			for i := 0; i < tc.firstObjCnt; i++ {
				require.NoError(wsize.Fill(&remotecommand.TerminalSize{Width: 1, Height: 1}))
			}
			if tc.closed {
				wsize.Close()
			}
			for i := 0; i < tc.secondObjCnt; i++ {
				err := wsize.Fill(&remotecommand.TerminalSize{Width: 1, Height: 1})
				assert.ErrorIs(err, tc.expectedErr)
			}
			wsize.Close()
		})
	}
}
