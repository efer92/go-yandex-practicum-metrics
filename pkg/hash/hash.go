// Package hash computes and verifies HMAC-SHA256 signatures of HTTP bodies.
package hash

import (
	"crypto/hmac"
	"crypto/sha256"
	"encoding/hex"
)

// Sign returns the hex-encoded HMAC-SHA256 of body keyed with key.
func Sign(body []byte, key string) string {
	h := hmac.New(sha256.New, []byte(key))
	h.Write(body)
	return hex.EncodeToString(h.Sum(nil))
}

// Equal compares two hex-encoded HMAC digests in constant time.
func Equal(a, b string) bool {
	aBytes, err1 := hex.DecodeString(a)
	bBytes, err2 := hex.DecodeString(b)
	if err1 != nil || err2 != nil {
		return false
	}
	return hmac.Equal(aBytes, bBytes)
}
