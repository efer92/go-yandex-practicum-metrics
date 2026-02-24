package repository

import (
	"os"
	"testing"

	"github.com/efer92/go-yandex-practicum-metrics/internal/model"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func tempFile(t *testing.T) string {
	t.Helper()
	f, err := os.CreateTemp("", "metrics-*.json")
	require.NoError(t, err)
	f.Close()
	t.Cleanup(func() { os.Remove(f.Name()) })
	return f.Name()
}

func TestFileBackedStorage_SaveAndLoad(t *testing.T) {
	path := tempFile(t)

	s := NewFileBackedStorage(path)
	s.UpdateGauge("Alloc", 1024.5)
	s.UpdateCounter("PollCount", 7)
	require.NoError(t, s.Save())

	s2 := NewFileBackedStorage(path)
	require.NoError(t, s2.Load())

	val, ok := s2.GetValue(model.Gauge, "Alloc")
	assert.True(t, ok)
	assert.Equal(t, "1024.5", val)

	cnt, ok := s2.GetValue(model.Counter, "PollCount")
	assert.True(t, ok)
	assert.Equal(t, "7", cnt)
}

func TestFileBackedStorage_LoadNonExistent(t *testing.T) {
	s := NewFileBackedStorage("/tmp/nonexistent-metrics-file.json")
	assert.NoError(t, s.Load())
}

func TestFileBackedStorage_LoadInvalidJSON(t *testing.T) {
	path := tempFile(t)
	os.WriteFile(path, []byte("not json"), 0644)

	s := NewFileBackedStorage(path)
	assert.Error(t, s.Load())
}

func TestFileBackedStorage_SaveCreatesFile(t *testing.T) {
	path := tempFile(t)
	os.Remove(path)

	s := NewFileBackedStorage(path)
	s.UpdateGauge("HeapAlloc", 512.0)
	require.NoError(t, s.Save())

	_, err := os.Stat(path)
	assert.NoError(t, err)
}

func TestFileBackedStorage_DelegatesStorage(t *testing.T) {
	s := NewFileBackedStorage("/tmp/test.json")

	s.UpdateGauge("Sys", 99.9)
	s.UpdateCounter("PollCount", 3)

	m, ok := s.GetMetric("Sys", model.Gauge)
	assert.True(t, ok)
	require.NotNil(t, m.Value)
	assert.Equal(t, 99.9, *m.Value)

	all := s.GetAllMetrics()
	assert.Contains(t, all, "gauge/Sys")
	assert.Contains(t, all, "counter/PollCount")
}
