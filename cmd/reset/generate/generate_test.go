package generate

import (
	"os"
	"path/filepath"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestGeneratePackage_SampleFixture(t *testing.T) {
	dir := filepath.Join("testdata", "sample")

	got, ok, err := generatePackage(dir)
	require.NoError(t, err)
	require.True(t, ok, "fixture must have marked structs")

	want, err := os.ReadFile(filepath.Join(dir, "want.go.txt"))
	require.NoError(t, err)

	assert.Equal(t, string(want), string(got))
}

func TestGeneratePackage_NoMarkers(t *testing.T) {
	// the generate package itself has no marked structs
	dir := "."
	_, ok, err := generatePackage(dir)
	require.NoError(t, err)
	assert.False(t, ok)
}

func TestRun_WritesFileAndIsIdempotent(t *testing.T) {
	dir := filepath.Join("testdata", "sample")
	out := filepath.Join(dir, GeneratedFile)

	// Clean up any generated file before and after.
	_ = os.Remove(out)
	t.Cleanup(func() { _ = os.Remove(out) })

	require.NoError(t, Run(dir))
	first, err := os.ReadFile(out)
	require.NoError(t, err)
	assert.NotEmpty(t, first)

	// Second run on the same dir must produce identical bytes.
	require.NoError(t, Run(dir))
	second, err := os.ReadFile(out)
	require.NoError(t, err)
	assert.Equal(t, string(first), string(second))
}
