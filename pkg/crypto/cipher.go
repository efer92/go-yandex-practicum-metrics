package crypto

import (
	"crypto/aes"
	"crypto/cipher"
	"crypto/rand"
	"crypto/rsa"
	"crypto/sha256"
	"encoding/binary"
	"fmt"
	"io"
)

// envelope layout:
//
//	[2 bytes BE: wrapped-key length]
//	[wrapped AES key (RSA-OAEP/SHA-256)]
//	[12 bytes nonce]
//	[ciphertext || GCM tag]
const (
	wrappedLenBytes = 2
	aesKeyBytes     = 32
	nonceBytes      = 12
)

// Encrypt wraps plaintext using AES-256-GCM with a fresh random key, and
// encrypts the AES key with RSA-OAEP under pub.
func Encrypt(pub *rsa.PublicKey, plaintext []byte) ([]byte, error) {
	aesKey := make([]byte, aesKeyBytes)
	if _, err := io.ReadFull(rand.Reader, aesKey); err != nil {
		return nil, fmt.Errorf("rand aes key: %w", err)
	}
	block, err := aes.NewCipher(aesKey)
	if err != nil {
		return nil, fmt.Errorf("aes new cipher: %w", err)
	}
	gcm, err := cipher.NewGCM(block)
	if err != nil {
		return nil, fmt.Errorf("aes gcm: %w", err)
	}
	nonce := make([]byte, gcm.NonceSize())
	if _, err := io.ReadFull(rand.Reader, nonce); err != nil {
		return nil, fmt.Errorf("rand nonce: %w", err)
	}
	ciphertext := gcm.Seal(nil, nonce, plaintext, nil)

	wrapped, err := rsa.EncryptOAEP(sha256.New(), rand.Reader, pub, aesKey, nil)
	if err != nil {
		return nil, fmt.Errorf("rsa wrap: %w", err)
	}
	if len(wrapped) > 0xFFFF {
		return nil, fmt.Errorf("wrapped key too large: %d", len(wrapped))
	}

	out := make([]byte, wrappedLenBytes+len(wrapped)+len(nonce)+len(ciphertext))
	binary.BigEndian.PutUint16(out[:wrappedLenBytes], uint16(len(wrapped)))
	copy(out[wrappedLenBytes:], wrapped)
	copy(out[wrappedLenBytes+len(wrapped):], nonce)
	copy(out[wrappedLenBytes+len(wrapped)+len(nonce):], ciphertext)
	return out, nil
}

// Decrypt reverses Encrypt. The envelope must have been produced by a peer
// using the public key paired with priv.
func Decrypt(priv *rsa.PrivateKey, envelope []byte) ([]byte, error) {
	if len(envelope) < wrappedLenBytes+nonceBytes {
		return nil, fmt.Errorf("envelope too short: %d bytes", len(envelope))
	}
	keyLen := int(binary.BigEndian.Uint16(envelope[:wrappedLenBytes]))
	if len(envelope) < wrappedLenBytes+keyLen+nonceBytes {
		return nil, fmt.Errorf("envelope truncated")
	}
	wrapped := envelope[wrappedLenBytes : wrappedLenBytes+keyLen]
	nonce := envelope[wrappedLenBytes+keyLen : wrappedLenBytes+keyLen+nonceBytes]
	ciphertext := envelope[wrappedLenBytes+keyLen+nonceBytes:]

	aesKey, err := rsa.DecryptOAEP(sha256.New(), rand.Reader, priv, wrapped, nil)
	if err != nil {
		return nil, fmt.Errorf("rsa unwrap: %w", err)
	}
	block, err := aes.NewCipher(aesKey)
	if err != nil {
		return nil, fmt.Errorf("aes new cipher: %w", err)
	}
	gcm, err := cipher.NewGCM(block)
	if err != nil {
		return nil, fmt.Errorf("aes gcm: %w", err)
	}
	plaintext, err := gcm.Open(nil, nonce, ciphertext, nil)
	if err != nil {
		return nil, fmt.Errorf("aes-gcm open: %w", err)
	}
	return plaintext, nil
}
