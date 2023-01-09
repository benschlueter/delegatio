package main

// PtyRequestPayload is the payload for a pty request.
type PtyRequestPayload struct {
	Term     string
	Columns  uint32
	Rows     uint32
	Width    uint32
	Height   uint32
	ModeList []byte
}

// ForwardTCPChannelOpenPayload is the payload for a forward-tcpip channel open request.
// RFC 4254 Section 7.2.
type ForwardTCPChannelOpenPayload struct {
	HostToConnect     string
	PortToConnect     uint32
	OriginatorAddress string
	OriginatorPort    uint32
}
