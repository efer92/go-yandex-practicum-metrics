package audit

import (
	"context"
	"encoding/json"
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestFileSink_AppendsJSONLine(t *testing.T) {
	dir := t.TempDir()
	path := filepath.Join(dir, "audit.log")
	sink := NewFileSink(path)

	ctx := context.Background()
	require.NoError(t, sink.Receive(ctx, Event{TS: 1, Metrics: []string{"A"}, IPAddress: "10.0.0.1"}))
	require.NoError(t, sink.Receive(ctx, Event{TS: 2, Metrics: []string{"B", "C"}, IPAddress: "10.0.0.2"}))

	data, err := os.ReadFile(path)
	require.NoError(t, err)

	lines := strings.Split(strings.TrimRight(string(data), "\n"), "\n")
	require.Len(t, lines, 2)

	var e1, e2 Event
	require.NoError(t, json.Unmarshal([]byte(lines[0]), &e1))
	require.NoError(t, json.Unmarshal([]byte(lines[1]), &e2))

	assert.Equal(t, Event{TS: 1, Metrics: []string{"A"}, IPAddress: "10.0.0.1"}, e1)
	assert.Equal(t, Event{TS: 2, Metrics: []string{"B", "C"}, IPAddress: "10.0.0.2"}, e2)
}

func TestFileSink_AppendsToExistingFile(t *testing.T) {
	dir := t.TempDir()
	path := filepath.Join(dir, "audit.log")
	require.NoError(t, os.WriteFile(path, []byte("pre-existing\n"), 0644))

	sink := NewFileSink(path)
	require.NoError(t, sink.Receive(context.Background(), Event{TS: 7, Metrics: []string{"M"}, IPAddress: "127.0.0.1"}))

	data, err := os.ReadFile(path)
	require.NoError(t, err)

	assert.True(t, strings.HasPrefix(string(data), "pre-existing\n"))
	assert.Contains(t, string(data), `"ts":7`)
}
