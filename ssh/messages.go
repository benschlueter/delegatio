/* SPDX-License-Identifier: AGPL-3.0-only
 * Copyright (c) Benedict Schlueter
 */

package main

// PtyRequestPayload is the payload for a pty request.
type PtyRequestPayload struct {
	Term         string
	WidthColumns uint32
	HeightRows   uint32
	WidthPixels  uint32
	HeightPixels uint32
	ModeList     []byte
}

// WindowChangeRequestPayload is the payload for a window-change request.
type WindowChangeRequestPayload struct {
	WidthColumns uint32
	HeightRows   uint32
	WidthPixels  uint32
	HeightPixels uint32
}

// SubsystemRequestPayload is the payload for a subsystem request.
type SubsystemRequestPayload struct {
	Subsystem string
}

// ForwardTCPChannelOpenPayload is the payload for a forward-tcpip channel open request.
// RFC 4254 Section 7.2.
type ForwardTCPChannelOpenPayload struct {
	HostToConnect     string
	PortToConnect     uint32
	OriginatorAddress string
	OriginatorPort    uint32
}
