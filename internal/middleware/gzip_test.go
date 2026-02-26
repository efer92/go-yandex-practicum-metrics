package middleware

import (
	"bytes"
	"compress/gzip"
	"io"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func gzipBody(t *testing.T, data []byte) *bytes.Buffer {
	t.Helper()
	var buf bytes.Buffer
	gz := gzip.NewWriter(&buf)
	_, err := gz.Write(data)
	require.NoError(t, err)
	require.NoError(t, gz.Close())
	return &buf
}

func TestGzipMiddleware_CompressesResponse(t *testing.T) {
	handler := GzipMiddleware(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		w.Write([]byte(`{"id":"Alloc","type":"gauge","value":1024}`))
	}))

	req := httptest.NewRequest(http.MethodGet, "/", nil)
	req.Header.Set("Accept-Encoding", "gzip")
	rec := httptest.NewRecorder()

	handler.ServeHTTP(rec, req)

	assert.Equal(t, "gzip", rec.Header().Get("Content-Encoding"))

	gr, err := gzip.NewReader(rec.Body)
	require.NoError(t, err)
	defer gr.Close()

	body, err := io.ReadAll(gr)
	require.NoError(t, err)
	assert.JSONEq(t, `{"id":"Alloc","type":"gauge","value":1024}`, string(body))
}

func TestGzipMiddleware_NoCompressWithoutHeader(t *testing.T) {
	handler := GzipMiddleware(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		w.Write([]byte(`{"id":"Alloc","type":"gauge","value":1024}`))
	}))

	req := httptest.NewRequest(http.MethodGet, "/", nil)
	// Accept-Encoding не задан
	rec := httptest.NewRecorder()

	handler.ServeHTTP(rec, req)

	assert.Empty(t, rec.Header().Get("Content-Encoding"))
	assert.JSONEq(t, `{"id":"Alloc","type":"gauge","value":1024}`, rec.Body.String())
}

func TestGzipMiddleware_DecompressesRequest(t *testing.T) {
	payload := `{"id":"Alloc","type":"gauge","value":1024}`
	compressed := gzipBody(t, []byte(payload))

	var received string
	handler := GzipMiddleware(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		body, err := io.ReadAll(r.Body)
		require.NoError(t, err)
		received = string(body)
		w.WriteHeader(http.StatusOK)
	}))

	req := httptest.NewRequest(http.MethodPost, "/update", compressed)
	req.Header.Set("Content-Encoding", "gzip")
	rec := httptest.NewRecorder()

	handler.ServeHTTP(rec, req)

	assert.Equal(t, http.StatusOK, rec.Code)
	assert.JSONEq(t, payload, received)
}

func TestGzipMiddleware_BadCompressedRequest(t *testing.T) {
	handler := GzipMiddleware(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusOK)
	}))

	req := httptest.NewRequest(http.MethodPost, "/update", bytes.NewBufferString("not gzip"))
	req.Header.Set("Content-Encoding", "gzip")
	rec := httptest.NewRecorder()

	handler.ServeHTTP(rec, req)

	assert.Equal(t, http.StatusBadRequest, rec.Code)
}

func TestGzipMiddleware_HTMLCompressed(t *testing.T) {
	handler := GzipMiddleware(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "text/html; charset=utf-8")
		w.Write([]byte("<html><body>metrics</body></html>"))
	}))

	req := httptest.NewRequest(http.MethodGet, "/", nil)
	req.Header.Set("Accept-Encoding", "gzip")
	rec := httptest.NewRecorder()

	handler.ServeHTTP(rec, req)

	assert.Equal(t, "gzip", rec.Header().Get("Content-Encoding"))

	gr, err := gzip.NewReader(rec.Body)
	require.NoError(t, err)
	defer gr.Close()

	body, err := io.ReadAll(gr)
	require.NoError(t, err)
	assert.Contains(t, string(body), "metrics")
}
