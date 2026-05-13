package audit

import (
	"context"
	"encoding/json"
	"io"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestHTTPObserver_PostsJSON(t *testing.T) {
	var received Event
	var receivedCT string
	var receivedMethod string

	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		receivedMethod = r.Method
		receivedCT = r.Header.Get("Content-Type")
		body, err := io.ReadAll(r.Body)
		require.NoError(t, err)
		require.NoError(t, json.Unmarshal(body, &received))
		w.WriteHeader(http.StatusOK)
	}))
	defer srv.Close()

	obs := NewHTTPObserver(srv.URL)
	e := Event{TS: 42, Metrics: []string{"Alloc", "Frees"}, IPAddress: "192.168.0.42"}
	require.NoError(t, obs.Receive(context.Background(), e))

	assert.Equal(t, http.MethodPost, receivedMethod)
	assert.Equal(t, "application/json", receivedCT)
	assert.Equal(t, e, received)
}

func TestHTTPObserver_NonOKReturnsError(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
		w.WriteHeader(http.StatusInternalServerError)
	}))
	defer srv.Close()

	obs := NewHTTPObserver(srv.URL)
	err := obs.Receive(context.Background(), Event{TS: 1, Metrics: []string{"X"}, IPAddress: "127.0.0.1"})

	assert.Error(t, err)
}
