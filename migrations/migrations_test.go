package migrations_test

import (
	"testing"

	"github.com/efer92/go-yandex-practicum-metrics/migrations"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestFS_ContainsMigrationFiles(t *testing.T) {
	entries, err := migrations.FS.ReadDir("sql")
	require.NoError(t, err, "папка sql должна существовать в embed.FS")
	assert.NotEmpty(t, entries, "папка sql не должна быть пустой")

	names := make([]string, 0, len(entries))
	for _, e := range entries {
		names = append(names, e.Name())
	}

	assert.Contains(t, names, "000001_create_metrics_table.up.sql")
	assert.Contains(t, names, "000001_create_metrics_table.down.sql")
}

func TestFS_MigrationFilesReadable(t *testing.T) {
	up, err := migrations.FS.ReadFile("sql/000001_create_metrics_table.up.sql")
	require.NoError(t, err)
	assert.NotEmpty(t, up, "up-миграция не должна быть пустой")

	down, err := migrations.FS.ReadFile("sql/000001_create_metrics_table.down.sql")
	require.NoError(t, err)
	assert.NotEmpty(t, down, "down-миграция не должна быть пустой")
}

func TestFS_UpMigrationContainsCreateTable(t *testing.T) {
	content, err := migrations.FS.ReadFile("sql/000001_create_metrics_table.up.sql")
	require.NoError(t, err)
	assert.Contains(t, string(content), "CREATE TABLE")
}

func TestFS_DownMigrationContainsDropTable(t *testing.T) {
	content, err := migrations.FS.ReadFile("sql/000001_create_metrics_table.down.sql")
	require.NoError(t, err)
	assert.Contains(t, string(content), "DROP TABLE")
}
