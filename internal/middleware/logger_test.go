package middleware

import (
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/stretchr/testify/assert"
	"go.uber.org/zap"
	"go.uber.org/zap/zaptest/observer"
)

func newTestLogger() (*zap.Logger, *observer.ObservedLogs) {
	core, logs := observer.New(zap.InfoLevel)
	return zap.New(core), logs
}

func TestLogger_LogsRequestAndResponse(t *testing.T) {
	log, logs := newTestLogger()

	handler := Logger(log)(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusCreated)
		w.Write([]byte("hello"))
	}))

	req := httptest.NewRequest(http.MethodPost, "/update/counter/test/1", nil)
	rec := httptest.NewRecorder()
	handler.ServeHTTP(rec, req)

	assert.Equal(t, 1, logs.Len())

	entry := logs.All()[0]
	assert.Equal(t, "request", entry.Message)

	fields := entry.ContextMap()
	assert.Equal(t, "/update/counter/test/1", fields["uri"])
	assert.Equal(t, http.MethodPost, fields["method"])
	assert.Equal(t, int64(http.StatusCreated), fields["status"])
	assert.Equal(t, int64(5), fields["size"]) // len("hello") == 5
}

func TestLogger_DefaultStatus200(t *testing.T) {
	log, logs := newTestLogger()

	handler := Logger(log)(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Write([]byte("ok"))
	}))

	req := httptest.NewRequest(http.MethodGet, "/", nil)
	rec := httptest.NewRecorder()
	handler.ServeHTTP(rec, req)

	entry := logs.All()[0]
	fields := entry.ContextMap()
	assert.Equal(t, int64(http.StatusOK), fields["status"])
	assert.Equal(t, int64(2), fields["size"]) // len("ok") == 2
}

func TestResponseWriter_Size(t *testing.T) {
	rec := httptest.NewRecorder()
	rw := &responseWriter{ResponseWriter: rec, status: http.StatusOK}

	n, err := rw.Write([]byte("hello world"))
	assert.NoError(t, err)
	assert.Equal(t, 11, n)
	assert.Equal(t, 11, rw.size)
}

func TestResponseWriter_Status(t *testing.T) {
	rec := httptest.NewRecorder()
	rw := &responseWriter{ResponseWriter: rec, status: http.StatusOK}

	rw.WriteHeader(http.StatusNotFound)
	assert.Equal(t, http.StatusNotFound, rw.status)
}
