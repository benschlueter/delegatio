/* SPDX-License-Identifier: AGPL-3.0-only
 * Copyright (c) Benedict Schlueter
 */

package utils

// EtcdCredentials contains the credentials for etcd.
type EtcdCredentials struct {
	PeerCertData []byte // "/etc/kubernetes/pki/etcd/peer.crt"
	KeyData      []byte // "/etc/kubernetes/pki/etcd/peer.key"
	CaCertData   []byte // "/etc/kubernetes/pki/etcd/server.crt"
}
