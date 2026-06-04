package buildinfo

import (
	"bytes"
	"testing"

	"github.com/stretchr/testify/assert"
)

func TestPrint_AllValuesPresent(t *testing.T) {
	var buf bytes.Buffer
	Print(&buf, "1.2.3", "2026-05-13", "abc1234")

	assert.Equal(t, "Build version: 1.2.3\nBuild date: 2026-05-13\nBuild commit: abc1234\n", buf.String())
}

func TestPrint_EmptyValuesUsePlaceholder(t *testing.T) {
	var buf bytes.Buffer
	Print(&buf, "", "", "")

	assert.Equal(t, "Build version: N/A\nBuild date: N/A\nBuild commit: N/A\n", buf.String())
}

func TestPrint_MixedValues(t *testing.T) {
	var buf bytes.Buffer
	Print(&buf, "1.0.0", "", "deadbeef")

	assert.Equal(t, "Build version: 1.0.0\nBuild date: N/A\nBuild commit: deadbeef\n", buf.String())
}
