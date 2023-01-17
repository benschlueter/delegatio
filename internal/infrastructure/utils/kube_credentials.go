/* SPDX-License-Identifier: AGPL-3.0-only
 * Copyright (c) Benedict Schlueter
 */

package utils

// EtcdCredentials contains the credentials for etcd.
type EtcdCredentials struct {
	PeerCertData []byte // self generated
	KeyData      []byte // self generated
	CaCertData   []byte // "/etc/kubernetes/pki/etcd/ca.crt"
}
