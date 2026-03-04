package cmd

import (
	"fmt"
	"os"
	"path/filepath"

	"github.com/spf13/cobra"

	"vine/config"
)

var dbCmd = &cobra.Command{
	Use:   "db",
	Short: "Manage vine databases",
}

var dbListCmd = &cobra.Command{
	Use:   "list",
	Short: "List local and global databases",
	RunE: func(cmd *cobra.Command, args []string) error {
		found := false

		// Check for local database.
		cwd, err := os.Getwd()
		if err != nil {
			return err
		}
		projectRoot, rootErr := config.FindProjectRoot(cwd)
		if rootErr == nil {
			cfg, cfgErr := config.Load(projectRoot)
			if cfgErr == nil {
				dbPath, pathErr := config.DatabasePath(projectRoot, cfg)
				if pathErr == nil {
					if _, err := os.Stat(dbPath); err == nil {
						fmt.Printf("Local (%s):\n  %s\n", cfg.Storage, dbPath)
						found = true
					}
				}
			}
		}

		// List global databases.
		globalDir, err := config.GlobalDatabasesDir()
		if err != nil {
			return err
		}
		entries, err := os.ReadDir(globalDir)
		if err != nil && !os.IsNotExist(err) {
			return fmt.Errorf("reading global databases dir: %w", err)
		}

		var globals []string
		for _, e := range entries {
			if !e.IsDir() && filepath.Ext(e.Name()) == ".db" {
				globals = append(globals, e.Name()[:len(e.Name())-3])
			}
		}

		if len(globals) > 0 {
			fmt.Printf("Global (~/.vine/databases/):\n")
			for _, name := range globals {
				fmt.Printf("  %s\n", name)
			}
			found = true
		}

		if !found {
			fmt.Println("No databases found.")
		}

		return nil
	},
}

var dbRenameCmd = &cobra.Command{
	Use:   "rename <new-name>",
	Short: "Rename the current project's database",
	Long:  "Renames the global database file and updates config. For global storage only — local databases derive their name from the directory.",
	Args:  cobra.ExactArgs(1),
	RunE: func(cmd *cobra.Command, args []string) error {
		newName := args[0]

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

		if cfg.Storage != config.StorageGlobal {
			return fmt.Errorf("rename is only supported for global storage (current: %s)", cfg.Storage)
		}

		if newName == cfg.Database {
			return fmt.Errorf("database is already named %q", newName)
		}

		oldPath, err := config.DatabasePath(projectRoot, cfg)
		if err != nil {
			return err
		}

		// Build the new path and check it doesn't already exist.
		globalDir, err := config.GlobalDatabasesDir()
		if err != nil {
			return err
		}
		newPath := filepath.Join(globalDir, newName+".db")
		if _, err := os.Stat(newPath); err == nil {
			return fmt.Errorf("database %q already exists at %s", newName, newPath)
		}

		// Rename all SQLite files.
		for _, ext := range dbSymlinkExtensions {
			src := oldPath + ext
			dst := newPath + ext
			if _, err := os.Stat(src); err == nil {
				if err := os.Rename(src, dst); err != nil {
					return fmt.Errorf("renaming %s: %w", src, err)
				}
			}
		}

		// Update config.
		oldName := cfg.Database
		cfg.Database = newName
		cfg.Name = newName
		if err := config.Save(projectRoot, cfg); err != nil {
			return err
		}

		// Recreate symlinks.
		if err := createDBSymlinks(projectRoot, cfg); err != nil {
			return fmt.Errorf("updating symlinks: %w", err)
		}

		PrintOutput(cmd, fmt.Sprintf("Renamed database %q → %q", oldName, newName), map[string]string{
			"old_name": oldName,
			"new_name": newName,
			"old_path": oldPath,
			"new_path": newPath,
		})
		return nil
	},
}

func init() {
	AddJSONFlag(dbListCmd)
	AddJSONFlag(dbRenameCmd)
	dbCmd.AddCommand(dbListCmd)
	dbCmd.AddCommand(dbRenameCmd)
	rootCmd.AddCommand(dbCmd)
}
