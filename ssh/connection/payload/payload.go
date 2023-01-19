/* SPDX-License-Identifier: AGPL-3.0-only
 * Copyright (c) Benedict Schlueter
 */

package payload

// PtyRequest is the payload for a pty request.
type PtyRequest struct {
	Term         string
	WidthColumns uint32
	HeightRows   uint32
	WidthPixels  uint32
	HeightPixels uint32
	ModeList     []byte
}

// WindowChangeRequest is the payload for a window-change request.
type WindowChangeRequest struct {
	WidthColumns uint32
	HeightRows   uint32
	WidthPixels  uint32
	HeightPixels uint32
}

// SubsystemRequest is the payload for a subsystem request.
type SubsystemRequest struct {
	Subsystem string
}

// ForwardTCPChannelOpen is the payload for a forward-tcpip channel open request.
// RFC 4254 Section 7.2.
type ForwardTCPChannelOpen struct {
	HostToConnect     string
	PortToConnect     uint32
	OriginatorAddress string
	OriginatorPort    uint32
}
