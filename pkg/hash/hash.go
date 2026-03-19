package hash

import (
	"crypto/hmac"
	"crypto/sha256"
	"encoding/hex"
)

func Sign(body []byte, key string) string {
	h := hmac.New(sha256.New, []byte(key))
	h.Write(body)
	return hex.EncodeToString(h.Sum(nil))
}

func Equal(a, b string) bool {
	aBytes, err1 := hex.DecodeString(a)
	bBytes, err2 := hex.DecodeString(b)
	if err1 != nil || err2 != nil {
		return false
	}
	return hmac.Equal(aBytes, bBytes)
}
