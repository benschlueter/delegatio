/* SPDX-License-Identifier: AGPL-3.0-only
 * Copyright (c) Benedict Schlueter
 */

package kubernetes

import (
	"crypto/rand"
	"crypto/rsa"
	"crypto/x509"
	"encoding/pem"
	"errors"

	"github.com/benschlueter/delegatio/internal/config"
)

// generateEtcdCertificate generates a new etcd certificate for the instance.
func (a *Bootstrapper) generateEtcdCertificate(caCert, caKey []byte) (*config.EtcdCredentials, error) {
	pemBlock, _ := pem.Decode(caCert)
	if pemBlock == nil {
		return nil, errors.New("no PEM data found in CA cert")
	}
	caCertX509, err := x509.ParseCertificate(pemBlock.Bytes)
	if err != nil {
		return nil, err
	}
	pemBlock, _ = pem.Decode(caKey)
	if pemBlock == nil {
		return nil, errors.New("no PEM data found in CA key")
	}
	caPrivateKey, err := x509.ParsePKCS1PrivateKey(pemBlock.Bytes)
	if err != nil {
		return nil, errors.New("cannot parse pkcs1 CA key pem block")
	}

	key, err := rsa.GenerateKey(rand.Reader, 4096)
	if err != nil {
		return nil, err
	}
	// Generate a pem block with the private key
	keyPem := pem.EncodeToMemory(&pem.Block{
		Type:  "RSA PRIVATE KEY",
		Bytes: x509.MarshalPKCS1PrivateKey(key),
	})
	a.log.Info("encode to mem")
	cert, err := x509.CreateCertificate(rand.Reader, caCertX509, caCertX509, &key.PublicKey, caPrivateKey)
	if err != nil {
		return nil, err
	}
	certPem := pem.EncodeToMemory(&pem.Block{
		Type:  "CERTIFICATE",
		Bytes: cert,
	})
	return &config.EtcdCredentials{
		PeerCertData: certPem,
		KeyData:      keyPem,
		CaCertData:   caCert,
	}, nil
}
