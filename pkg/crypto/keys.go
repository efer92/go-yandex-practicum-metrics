// Package crypto provides RSA key loading and a hybrid RSA-OAEP + AES-GCM
// envelope used to encrypt agent-to-server request bodies.
package crypto

import (
	"crypto/rsa"
	"crypto/x509"
	"encoding/pem"
	"fmt"
	"os"
)

// LoadPublicKey reads a PEM-encoded RSA public key from path.
// Both PKIX (BEGIN PUBLIC KEY) and PKCS#1 (BEGIN RSA PUBLIC KEY) are accepted.
func LoadPublicKey(path string) (*rsa.PublicKey, error) {
	data, err := os.ReadFile(path)
	if err != nil {
		return nil, fmt.Errorf("read public key %q: %w", path, err)
	}
	block, _ := pem.Decode(data)
	if block == nil {
		return nil, fmt.Errorf("no PEM block in %q", path)
	}

	if pub, err := x509.ParsePKIXPublicKey(block.Bytes); err == nil {
		rsaPub, ok := pub.(*rsa.PublicKey)
		if !ok {
			return nil, fmt.Errorf("not an RSA public key: %T", pub)
		}
		return rsaPub, nil
	}
	if rsaPub, err := x509.ParsePKCS1PublicKey(block.Bytes); err == nil {
		return rsaPub, nil
	}
	return nil, fmt.Errorf("parse public key %q: unsupported format", path)
}

// LoadPrivateKey reads a PEM-encoded RSA private key from path.
// Both PKCS#1 (BEGIN RSA PRIVATE KEY) and PKCS#8 (BEGIN PRIVATE KEY) are accepted.
func LoadPrivateKey(path string) (*rsa.PrivateKey, error) {
	data, err := os.ReadFile(path)
	if err != nil {
		return nil, fmt.Errorf("read private key %q: %w", path, err)
	}
	block, _ := pem.Decode(data)
	if block == nil {
		return nil, fmt.Errorf("no PEM block in %q", path)
	}

	if priv, err := x509.ParsePKCS1PrivateKey(block.Bytes); err == nil {
		return priv, nil
	}
	if priv, err := x509.ParsePKCS8PrivateKey(block.Bytes); err == nil {
		rsaPriv, ok := priv.(*rsa.PrivateKey)
		if !ok {
			return nil, fmt.Errorf("not an RSA private key: %T", priv)
		}
		return rsaPriv, nil
	}
	return nil, fmt.Errorf("parse private key %q: unsupported format", path)
}
