// Package httpconst declares HTTP-protocol constants (header names) shared
// between the agent client and server middleware. Keeping them here breaks
// the cycle where the agent would otherwise have to import internal/middleware
// just to spell a single header.
package httpconst

// HeaderHashSHA256 carries the HMAC-SHA256 signature of the request/response body.
const HeaderHashSHA256 = "HashSHA256"

// HeaderCryptoEncrypted marks a request whose body is RSA-OAEP+AES-GCM encrypted.
const HeaderCryptoEncrypted = "X-Crypto-Encrypted"

// HeaderXRealIP carries the agent host's IP address; the server validates it
// against a trusted CIDR subnet when one is configured.
const HeaderXRealIP = "X-Real-IP"
