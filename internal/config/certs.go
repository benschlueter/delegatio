/* SPDX-License-Identifier: AGPL-3.0-only
 * Copyright (c) Benedict Schlueter
 */

package config

import (
	"crypto"
	"crypto/rand"
	"crypto/rsa"
	"crypto/tls"
	"crypto/x509"
	"crypto/x509/pkix"
	"encoding/pem"
	"fmt"
	"math/big"
	"net"
	"time"

	"github.com/benschlueter/delegatio/internal/config/definitions"
)

func getIPAddr() (string, error) {
	conn, err := net.Dial("udp", "8.8.8.8:80")
	if err != nil {
		return "", err
	}
	defer conn.Close()

	localAddr := conn.LocalAddr().(*net.UDPAddr)

	return localAddr.IP.String(), nil
}

// GenerateTLSConfigServer generates a TLS configuration for the server.
func GenerateTLSConfigServer() (*tls.Config, error) {
	rootkey, err := rootKeyFile.ReadFile("rootkey.pem")
	if err != nil {
		return nil, err
	}
	caKeyBlock, _ := pem.Decode(rootkey)
	if caKeyBlock == nil || caKeyBlock.Type != "PRIVATE KEY" {
		return nil, fmt.Errorf("failed to decode CA key, unsupported type: %v", caKeyBlock.Type)
	}
	caPrivateKey, err := x509.ParsePKCS8PrivateKey(caKeyBlock.Bytes)
	if err != nil {
		return nil, err
	}
	publicKeySigner, ok := caPrivateKey.(crypto.Signer)
	if !ok {
		return nil, fmt.Errorf("failed to cast private key to crypto.Signer")
	}
	caPublicKey := publicKeySigner.Public()

	// Generate a root CA certificate
	caTemplate := &x509.Certificate{
		SerialNumber: big.NewInt(1),
		Subject: pkix.Name{
			CommonName: "Delegatio Root CA",
		},
		KeyUsage:              x509.KeyUsageCertSign | x509.KeyUsageDigitalSignature,
		BasicConstraintsValid: true,
		IsCA:                  true,
		NotAfter:              time.Now().Add(365 * 24 * time.Hour), // 1 year
	}
	caCertBytes, err := x509.CreateCertificate(rand.Reader, caTemplate, caTemplate, caPublicKey, caPrivateKey)
	if err != nil {
		return nil, err
	}

	// Generate a new client key pair
	clientPrivateKey, err := rsa.GenerateKey(rand.Reader, 4096)
	if err != nil {
		return nil, err
	}

	ipAddr, err := getIPAddr()
	if err != nil {
		return nil, err
	}

	// Sign the CSR with the root CA certificate and key to generate a new client certificate
	clientCertTemplate := &x509.Certificate{
		SerialNumber: big.NewInt(0xB00BFFFF),
		Subject: pkix.Name{
			CommonName: "Delegatio Client",
		},
		NotBefore:   time.Now(),
		NotAfter:    time.Now().Add(365 * 24 * time.Hour), // 1 year
		KeyUsage:    x509.KeyUsageKeyEncipherment | x509.KeyUsageDigitalSignature | x509.KeyUsageKeyAgreement | x509.KeyUsageDataEncipherment,
		ExtKeyUsage: []x509.ExtKeyUsage{x509.ExtKeyUsageServerAuth},
		IPAddresses: []net.IP{net.ParseIP(ipAddr), net.ParseIP(definitions.NetworkXMLConfig.IPs[0].Address)},
		DNSNames:    []string{"localhost", "delegatio-master-0", "delegatio-master-1", "delegatio-master-2", "delegatio-worker-0", "delegatio-worker-1", "delegatio-worker-2"},
	}
	clientCertBytes, err := x509.CreateCertificate(rand.Reader, clientCertTemplate, caTemplate, &clientPrivateKey.PublicKey, caPrivateKey)
	if err != nil {
		return nil, err
	}

	// Load the new client certificate and key pair into the TLS configuration
	clientCert := tls.Certificate{
		Certificate: [][]byte{clientCertBytes},
		PrivateKey:  clientPrivateKey,
	}

	// Create a cert pool and add the root CA certificate
	certPool := x509.NewCertPool()
	certPool.AppendCertsFromPEM(pem.EncodeToMemory(&pem.Block{Type: "CERTIFICATE", Bytes: caCertBytes}))

	tlsConfig := &tls.Config{
		ClientAuth:   tls.RequireAndVerifyClientCert,
		ClientCAs:    certPool,
		MinVersion:   tls.VersionTLS13,
		Certificates: []tls.Certificate{clientCert},
	}

	return tlsConfig, nil
}

// GenerateTLSConfigClient generates a TLS configuration for the client.
func GenerateTLSConfigClient() (*tls.Config, error) {
	rootkey, err := rootKeyFile.ReadFile("rootkey.pem")
	if err != nil {
		return nil, err
	}
	caKeyBlock, _ := pem.Decode(rootkey)
	if caKeyBlock == nil || caKeyBlock.Type != "PRIVATE KEY" {
		return nil, fmt.Errorf("failed to decode CA key, unsupported type: %v", caKeyBlock.Type)
	}
	caPrivateKey, err := x509.ParsePKCS8PrivateKey(caKeyBlock.Bytes)
	if err != nil {
		return nil, err
	}
	publicKeySigner, ok := caPrivateKey.(crypto.Signer)
	if !ok {
		return nil, fmt.Errorf("failed to cast private key to crypto.Signer")
	}
	caPublicKey := publicKeySigner.Public()

	// Generate a root CA certificate
	caTemplate := &x509.Certificate{
		SerialNumber: big.NewInt(1),
		Subject: pkix.Name{
			CommonName: "Delegatio Root CA",
		},
		KeyUsage:              x509.KeyUsageCertSign | x509.KeyUsageDigitalSignature,
		BasicConstraintsValid: true,
		IsCA:                  true,
		NotAfter:              time.Now().Add(365 * 24 * time.Hour), // 1 year
	}
	caCertBytes, err := x509.CreateCertificate(rand.Reader, caTemplate, caTemplate, caPublicKey, caPrivateKey)
	if err != nil {
		return nil, err
	}

	// Generate a new client key pair
	clientPrivateKey, err := rsa.GenerateKey(rand.Reader, 4096)
	if err != nil {
		return nil, err
	}

	ipAddr, err := getIPAddr()
	if err != nil {
		return nil, err
	}

	// Sign the CSR with the root CA certificate and key to generate a new client certificate
	clientCertTemplate := &x509.Certificate{
		SerialNumber: big.NewInt(0xB00BFFFF),
		Subject: pkix.Name{
			CommonName: "Delegatio Client",
		},
		NotBefore:   time.Now(),
		NotAfter:    time.Now().Add(365 * 24 * time.Hour), // 1 year
		KeyUsage:    x509.KeyUsageKeyEncipherment | x509.KeyUsageDigitalSignature | x509.KeyUsageKeyAgreement | x509.KeyUsageDataEncipherment,
		ExtKeyUsage: []x509.ExtKeyUsage{x509.ExtKeyUsageClientAuth},
		IPAddresses: []net.IP{net.ParseIP(ipAddr), net.ParseIP(definitions.NetworkXMLConfig.IPs[0].Address)},
		DNSNames:    []string{"localhost", "delegatio-master-0", "delegatio-master-1", "delegatio-master-2", "delegatio-worker-0", "delegatio-worker-1", "delegatio-worker-2"},
	}
	clientCertBytes, err := x509.CreateCertificate(rand.Reader, clientCertTemplate, caTemplate, &clientPrivateKey.PublicKey, caPrivateKey)
	if err != nil {
		return nil, err
	}

	// Load the new client certificate and key pair into the TLS configuration
	clientCert := tls.Certificate{
		Certificate: [][]byte{clientCertBytes},
		PrivateKey:  clientPrivateKey,
	}

	// Create a cert pool and add the root CA certificate
	certPool := x509.NewCertPool()
	certPool.AppendCertsFromPEM(pem.EncodeToMemory(&pem.Block{Type: "CERTIFICATE", Bytes: caCertBytes}))

	tlsConfig := &tls.Config{
		RootCAs:      certPool,
		MinVersion:   tls.VersionTLS13,
		Certificates: []tls.Certificate{clientCert},
	}

	return tlsConfig, nil
}
