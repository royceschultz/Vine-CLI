package cmd

import (
	"fmt"
	"os"
	"path/filepath"

	"github.com/spf13/cobra"

	"vine/config"
)

// dbSymlinkExtensions are the SQLite file extensions to symlink.
var dbSymlinkExtensions = []string{"", "-wal", "-shm"}

// createDBSymlinks creates symlinks in .vine/ pointing to the global database files.
// Returns an error if the config is not using global storage.
func createDBSymlinks(projectRoot string, cfg *config.Config) error {
	if cfg.Storage != config.StorageGlobal {
		return fmt.Errorf("symlinks are only needed for global storage (current: %s)", cfg.Storage)
	}

	dbPath, err := config.DatabasePath(projectRoot, cfg)
	if err != nil {
		return err
	}

	dotVine := filepath.Join(projectRoot, config.DotVineDir)
	for _, ext := range dbSymlinkExtensions {
		link := filepath.Join(dotVine, "vine.db"+ext)
		target := dbPath + ext
		os.Remove(link) // remove stale link
		if err := os.Symlink(target, link); err != nil {
			return fmt.Errorf("creating symlink %s → %s: %w", link, target, err)
		}
	}
	return nil
}

// checkDBSymlinks returns true if all expected symlinks exist and point to the right targets.
func checkDBSymlinks(projectRoot string, cfg *config.Config) (ok bool, details string) {
	if cfg.Storage != config.StorageGlobal {
		return true, "local storage, no symlinks needed"
	}

	dbPath, err := config.DatabasePath(projectRoot, cfg)
	if err != nil {
		return false, fmt.Sprintf("cannot resolve database path: %v", err)
	}

	dotVine := filepath.Join(projectRoot, config.DotVineDir)
	for _, ext := range dbSymlinkExtensions {
		link := filepath.Join(dotVine, "vine.db"+ext)
		target, err := os.Readlink(link)
		if err != nil {
			return false, fmt.Sprintf("missing symlink: vine.db%s", ext)
		}
		if target != dbPath+ext {
			return false, fmt.Sprintf("vine.db%s points to %s, expected %s", ext, target, dbPath+ext)
		}
	}
	return true, "symlinks point to global database"
}

var symlinkCmd = &cobra.Command{
	Use:   "symlink",
	Short: "Manage database symlinks for file watching",
}

var symlinkCreateCmd = &cobra.Command{
	Use:   "create",
	Short: "Create symlinks from .vine/ to the global database",
	Long:  "Creates vine.db, vine.db-wal, and vine.db-shm symlinks in .vine/ pointing to the global database. Only works with global storage.",
	RunE: func(cmd *cobra.Command, args []string) error {
		cwd, err := os.Getwd()
		if err != nil {
			return err
		}
		projectRoot, err := config.FindProjectRoot(cwd)
		if err != nil {
			return err
		}
		cfg, err := config.Load(projectRoot)
		if err != nil {
			return err
		}

		if err := createDBSymlinks(projectRoot, cfg); err != nil {
			return err
		}

		dbPath, _ := config.DatabasePath(projectRoot, cfg)
		PrintOutput(cmd, fmt.Sprintf("Symlinks created in .vine/ → %s", dbPath), map[string]string{
			"target": dbPath,
		})
		return nil
	},
}

func init() {
	AddJSONFlag(symlinkCreateCmd)
	symlinkCmd.AddCommand(symlinkCreateCmd)
	rootCmd.AddCommand(symlinkCmd)
}
