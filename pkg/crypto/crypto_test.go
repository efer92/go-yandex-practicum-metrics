package crypto

import (
	"bytes"
	"crypto/rand"
	"crypto/rsa"
	"crypto/x509"
	"encoding/pem"
	"os"
	"path/filepath"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func generateKey(t *testing.T) *rsa.PrivateKey {
	t.Helper()
	priv, err := rsa.GenerateKey(rand.Reader, 2048)
	require.NoError(t, err)
	return priv
}

func writePEM(t *testing.T, path, typ string, der []byte) {
	t.Helper()
	require.NoError(t, os.WriteFile(path, pem.EncodeToMemory(&pem.Block{Type: typ, Bytes: der}), 0600))
}

func TestEncryptDecrypt_RoundTrip(t *testing.T) {
	priv := generateKey(t)
	plaintext := []byte("hello, world — encrypted with RSA-OAEP + AES-256-GCM")

	envelope, err := Encrypt(&priv.PublicKey, plaintext)
	require.NoError(t, err)
	require.NotEqual(t, plaintext, envelope)

	got, err := Decrypt(priv, envelope)
	require.NoError(t, err)
	assert.Equal(t, plaintext, got)
}

func TestEncryptDecrypt_LargePayload(t *testing.T) {
	priv := generateKey(t)
	plaintext := bytes.Repeat([]byte("AB"), 10_000) // 20 KB — well past RSA's direct limit

	envelope, err := Encrypt(&priv.PublicKey, plaintext)
	require.NoError(t, err)

	got, err := Decrypt(priv, envelope)
	require.NoError(t, err)
	assert.Equal(t, plaintext, got)
}

func TestDecrypt_TamperedCiphertextFails(t *testing.T) {
	priv := generateKey(t)
	envelope, err := Encrypt(&priv.PublicKey, []byte("payload"))
	require.NoError(t, err)

	envelope[len(envelope)-1] ^= 0xFF // flip a bit in the tag
	_, err = Decrypt(priv, envelope)
	assert.Error(t, err)
}

func TestLoadKeys_PKIXAndPKCS1(t *testing.T) {
	dir := t.TempDir()
	priv := generateKey(t)

	privPath := filepath.Join(dir, "priv.pem")
	pubPath := filepath.Join(dir, "pub.pem")

	writePEM(t, privPath, "RSA PRIVATE KEY", x509.MarshalPKCS1PrivateKey(priv))
	pubDER, err := x509.MarshalPKIXPublicKey(&priv.PublicKey)
	require.NoError(t, err)
	writePEM(t, pubPath, "PUBLIC KEY", pubDER)

	loadedPriv, err := LoadPrivateKey(privPath)
	require.NoError(t, err)
	loadedPub, err := LoadPublicKey(pubPath)
	require.NoError(t, err)

	envelope, err := Encrypt(loadedPub, []byte("ping"))
	require.NoError(t, err)
	plaintext, err := Decrypt(loadedPriv, envelope)
	require.NoError(t, err)
	assert.Equal(t, []byte("ping"), plaintext)
}

func TestLoad_MissingFile(t *testing.T) {
	_, err := LoadPublicKey("/no/such/file")
	assert.Error(t, err)
	_, err = LoadPrivateKey("/no/such/file")
	assert.Error(t, err)
}
