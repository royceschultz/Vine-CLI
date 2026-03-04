package cmd

import (
	"os"
	"path/filepath"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"vine/config"
)

func setupGlobalProject(t *testing.T) (projectRoot string, cfg *config.Config) {
	t.Helper()

	projectRoot = t.TempDir()
	globalDir := t.TempDir()

	dbName := "testdb"
	dbPath := filepath.Join(globalDir, dbName+".db")

	// Create a fake database file.
	require.NoError(t, os.WriteFile(dbPath, []byte("fake"), 0o644))

	cfg = &config.Config{
		Name:     dbName,
		Storage:  config.StorageGlobal,
		Database: dbName,
	}

	// Save config.
	require.NoError(t, config.Save(projectRoot, cfg))

	// Patch DatabasePath to use our temp dir instead of ~/.vine.
	// We can't easily do that, so instead we'll test the symlink helpers
	// using a local-storage config (for rejection) and by manually
	// creating the expected structure.

	return projectRoot, cfg
}

func TestCreateDBSymlinks_RejectsLocal(t *testing.T) {
	dir := t.TempDir()
	cfg := &config.Config{Storage: config.StorageLocal}
	require.NoError(t, config.Save(dir, cfg))

	err := createDBSymlinks(dir, cfg)
	assert.Error(t, err)
	assert.Contains(t, err.Error(), "only needed for global storage")
}

func TestCheckDBSymlinks_LocalStorage(t *testing.T) {
	dir := t.TempDir()
	cfg := &config.Config{Storage: config.StorageLocal}

	ok, detail := checkDBSymlinks(dir, cfg)
	assert.True(t, ok)
	assert.Contains(t, detail, "no symlinks needed")
}

func TestCreateAndCheckDBSymlinks(t *testing.T) {
	projectRoot := t.TempDir()

	// Create .vine directory.
	dotVine := filepath.Join(projectRoot, config.DotVineDir)
	require.NoError(t, os.MkdirAll(dotVine, 0o755))

	cfg := &config.Config{
		Name:     "testproj",
		Storage:  config.StorageGlobal,
		Database: "testproj",
	}

	// Create symlinks (targets won't exist, but symlinks are still created).
	err := createDBSymlinks(projectRoot, cfg)
	require.NoError(t, err)

	// Verify all three symlinks exist.
	for _, ext := range dbSymlinkExtensions {
		link := filepath.Join(dotVine, "vine.db"+ext)
		target, err := os.Readlink(link)
		require.NoError(t, err, "symlink vine.db%s should exist", ext)
		assert.Contains(t, target, "testproj.db"+ext)
	}

	// checkDBSymlinks should pass.
	ok, _ := checkDBSymlinks(projectRoot, cfg)
	assert.True(t, ok)
}

func TestCheckDBSymlinks_Missing(t *testing.T) {
	projectRoot := t.TempDir()
	dotVine := filepath.Join(projectRoot, config.DotVineDir)
	require.NoError(t, os.MkdirAll(dotVine, 0o755))

	cfg := &config.Config{
		Name:     "testproj",
		Storage:  config.StorageGlobal,
		Database: "testproj",
	}

	// No symlinks created — check should fail.
	ok, detail := checkDBSymlinks(projectRoot, cfg)
	assert.False(t, ok)
	assert.Contains(t, detail, "missing symlink")
}

func TestCheckDBSymlinks_WrongTarget(t *testing.T) {
	projectRoot := t.TempDir()
	dotVine := filepath.Join(projectRoot, config.DotVineDir)
	require.NoError(t, os.MkdirAll(dotVine, 0o755))

	cfg := &config.Config{
		Name:     "testproj",
		Storage:  config.StorageGlobal,
		Database: "testproj",
	}

	// Create symlinks pointing to wrong targets.
	for _, ext := range dbSymlinkExtensions {
		link := filepath.Join(dotVine, "vine.db"+ext)
		os.Symlink("/wrong/path/db"+ext, link)
	}

	ok, detail := checkDBSymlinks(projectRoot, cfg)
	assert.False(t, ok)
	assert.Contains(t, detail, "points to")
}

func TestCreateDBSymlinks_OverwritesStale(t *testing.T) {
	projectRoot := t.TempDir()
	dotVine := filepath.Join(projectRoot, config.DotVineDir)
	require.NoError(t, os.MkdirAll(dotVine, 0o755))

	cfg := &config.Config{
		Name:     "testproj",
		Storage:  config.StorageGlobal,
		Database: "testproj",
	}

	// Create stale symlinks.
	for _, ext := range dbSymlinkExtensions {
		link := filepath.Join(dotVine, "vine.db"+ext)
		os.Symlink("/old/path"+ext, link)
	}

	// Recreate should overwrite them.
	err := createDBSymlinks(projectRoot, cfg)
	require.NoError(t, err)

	ok, _ := checkDBSymlinks(projectRoot, cfg)
	assert.True(t, ok)
}
