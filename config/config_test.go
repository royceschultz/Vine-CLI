package config

import (
	"os"
	"path/filepath"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestSaveAndLoad(t *testing.T) {
	dir := t.TempDir()

	cfg := &Config{
		Name:    "myproject",
		Storage: StorageLocal,
	}
	require.NoError(t, Save(dir, cfg))

	loaded, err := Load(dir)
	require.NoError(t, err)
	assert.Equal(t, "myproject", loaded.Name)
	assert.Equal(t, StorageLocal, loaded.Storage)
}

func TestSaveAndLoad_Global(t *testing.T) {
	dir := t.TempDir()

	cfg := &Config{
		Name:     "shared",
		Storage:  StorageGlobal,
		Database: "shared",
	}
	require.NoError(t, Save(dir, cfg))

	loaded, err := Load(dir)
	require.NoError(t, err)
	assert.Equal(t, StorageGlobal, loaded.Storage)
	assert.Equal(t, "shared", loaded.Database)
}

func TestLoad_MissingFile(t *testing.T) {
	dir := t.TempDir()

	_, err := Load(dir)
	assert.Error(t, err)
	assert.Contains(t, err.Error(), "reading config")
}

func TestLoad_InvalidJSON(t *testing.T) {
	dir := t.TempDir()
	configDir := filepath.Join(dir, DotVineDir)
	require.NoError(t, os.MkdirAll(configDir, 0o755))
	require.NoError(t, os.WriteFile(filepath.Join(configDir, ConfigFile), []byte("{invalid"), 0o644))

	_, err := Load(dir)
	assert.Error(t, err)
	assert.Contains(t, err.Error(), "parsing config")
}

func TestSave_CreatesDirectory(t *testing.T) {
	dir := t.TempDir()

	// .vine/ doesn't exist yet — Save should create it.
	cfg := &Config{Name: "test", Storage: StorageLocal}
	require.NoError(t, Save(dir, cfg))

	info, err := os.Stat(filepath.Join(dir, DotVineDir))
	require.NoError(t, err)
	assert.True(t, info.IsDir())
}

func TestDatabasePath_Local(t *testing.T) {
	cfg := &Config{Storage: StorageLocal}
	path, err := DatabasePath("/projects/foo", cfg)
	require.NoError(t, err)
	assert.Equal(t, "/projects/foo/.vine/vine.db", path)
}

func TestDatabasePath_Global(t *testing.T) {
	cfg := &Config{Storage: StorageGlobal, Database: "mydb"}
	path, err := DatabasePath("/projects/foo", cfg)
	require.NoError(t, err)

	home, _ := os.UserHomeDir()
	expected := filepath.Join(home, ".vine", "databases", "mydb.db")
	assert.Equal(t, expected, path)
}

func TestDatabasePath_GlobalMissingName(t *testing.T) {
	cfg := &Config{Storage: StorageGlobal}
	_, err := DatabasePath("/projects/foo", cfg)
	assert.Error(t, err)
	assert.Contains(t, err.Error(), "database name")
}

func TestDatabasePath_UnknownStorage(t *testing.T) {
	cfg := &Config{Storage: "s3"}
	_, err := DatabasePath("/projects/foo", cfg)
	assert.Error(t, err)
	assert.Contains(t, err.Error(), "unknown storage mode")
}

func TestFindProjectRoot(t *testing.T) {
	dir := t.TempDir()
	vineDir := filepath.Join(dir, DotVineDir)
	require.NoError(t, os.MkdirAll(vineDir, 0o755))

	root, err := FindProjectRoot(dir)
	require.NoError(t, err)
	assert.Equal(t, dir, root)
}

func TestFindProjectRoot_WalksUp(t *testing.T) {
	dir := t.TempDir()
	vineDir := filepath.Join(dir, DotVineDir)
	require.NoError(t, os.MkdirAll(vineDir, 0o755))

	subdir := filepath.Join(dir, "src", "pkg")
	require.NoError(t, os.MkdirAll(subdir, 0o755))

	root, err := FindProjectRoot(subdir)
	require.NoError(t, err)
	assert.Equal(t, dir, root)
}

func TestFindProjectRoot_NotFound(t *testing.T) {
	dir := t.TempDir()

	_, err := FindProjectRoot(dir)
	assert.Error(t, err)
	assert.Contains(t, err.Error(), "not a vine project")
}

func TestProjectName_FromConfig(t *testing.T) {
	assert.Equal(t, "myproject", ProjectName("/tmp/dir", &Config{Name: "myproject"}))
}

func TestProjectName_FromDatabase(t *testing.T) {
	assert.Equal(t, "shared-db", ProjectName("/tmp/dir", &Config{Database: "shared-db"}))
}

func TestProjectName_FromDirectory(t *testing.T) {
	assert.Equal(t, "dir", ProjectName("/tmp/dir", &Config{}))
}

func TestProjectName_Precedence(t *testing.T) {
	cfg := &Config{Name: "name", Database: "db"}
	assert.Equal(t, "name", ProjectName("/tmp/dir", cfg))
}
