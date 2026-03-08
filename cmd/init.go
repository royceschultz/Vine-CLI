package cmd

import (
	"bufio"
	"fmt"
	"os"
	"path/filepath"
	"strings"

	"github.com/spf13/cobra"

	"vine/config"
	"vine/store"
	"vine/tui"
)

var initCmd = &cobra.Command{
	Use:   "init",
	Short: "Create config for a new project",
	RunE: func(cmd *cobra.Command, args []string) error {
		cwd, err := os.Getwd()
		if err != nil {
			return err
		}

		dotVine := filepath.Join(cwd, config.DotVineDir)
		if _, err := os.Stat(dotVine); err == nil {
			return fmt.Errorf("already initialized: %s exists", dotVine)
		}

		storageFlag, _ := cmd.Flags().GetString("storage")
		nameFlag, _ := cmd.Flags().GetString("name")

		var cfg config.Config

		gitTrackedFlag, _ := cmd.Flags().GetString("git-tracked")

		if storageFlag != "" {
			cfg.Storage = config.StorageMode(storageFlag)
			if cfg.Storage == config.StorageGlobal {
				if nameFlag == "" {
					return fmt.Errorf("--name is required when --storage=global")
				}
				cfg.Database = nameFlag
				cfg.Name = nameFlag
			} else {
				cfg.Name = filepath.Base(cwd)
			}
			cfg.GitTracked = gitTrackedFlag == "true" || gitTrackedFlag == "yes"
		} else {
			reader := bufio.NewReader(os.Stdin)

			fmt.Println("Welcome to vine!")
			fmt.Println()
			fmt.Println("Where should vine store its database?")
			fmt.Println("  [l] Local  - .vine/vine.db in this directory")
			fmt.Println("  [g] Global - ~/.vine/databases/<name>.db (shareable across worktrees)")
			fmt.Print("Choice (l/g) [l]: ")

			choice, _ := reader.ReadString('\n')
			choice = strings.TrimSpace(strings.ToLower(choice))

			if choice == "g" || choice == "global" {
				cfg.Storage = config.StorageGlobal

				existing := listGlobalDatabases()
				result, err := tui.SelectDatabase(existing)
				if err != nil {
					return err
				}

				cfg.Database = result.Name
				cfg.Name = result.Name
			} else {
				cfg.Storage = config.StorageLocal
				cfg.Name = filepath.Base(cwd)
			}

			fmt.Println()
			fmt.Println("Should vine tasks be committed to git?")
			fmt.Println("  [y] Yes - track .vine/ in version control")
			fmt.Println("  [n] No  - add .vine/.gitignore to exclude from git")
			fmt.Print("Choice (y/n) [n]: ")

			gitChoice, _ := reader.ReadString('\n')
			gitChoice = strings.TrimSpace(strings.ToLower(gitChoice))
			cfg.GitTracked = gitChoice == "y" || gitChoice == "yes"
		}

		if err := config.Save(cwd, &cfg); err != nil {
			return err
		}

		if !cfg.GitTracked {
			if err := config.WriteGitIgnore(cwd); err != nil {
				return fmt.Errorf("writing .gitignore: %w", err)
			}
		}

		dbPath, err := config.DatabasePath(cwd, &cfg)
		if err != nil {
			return err
		}

		// Check if the database already exists before opening it.
		_, statErr := os.Stat(dbPath)
		dbExisted := statErr == nil

		s, err := store.OpenPath(dbPath)
		if err != nil {
			return fmt.Errorf("creating database: %w", err)
		}
		defer s.Close()

		// For global storage, create symlinks so file watchers can detect changes.
		if cfg.Storage == config.StorageGlobal {
			if err := createDBSymlinks(cwd, &cfg); err != nil {
				return err
			}
		}

		fmt.Printf("Initialized vine project in %s\n", dotVine)
		if dbExisted {
			fmt.Printf("Database: %s (joined existing)\n", dbPath)
		} else {
			fmt.Printf("Database: %s (created)\n", dbPath)
		}

		return nil
	},
}

// listGlobalDatabases returns the names of all databases in ~/.vine/databases/.
func listGlobalDatabases() []string {
	globalDir, err := config.GlobalDatabasesDir()
	if err != nil {
		return nil
	}
	entries, err := os.ReadDir(globalDir)
	if err != nil {
		return nil
	}
	var names []string
	for _, e := range entries {
		if !e.IsDir() && filepath.Ext(e.Name()) == ".db" {
			names = append(names, e.Name()[:len(e.Name())-3])
		}
	}
	return names
}

func init() {
	initCmd.Flags().String("storage", "", "storage mode: local or global (skips interactive prompt)")
	initCmd.Flags().String("name", "", "database name (required when --storage=global)")
	initCmd.Flags().String("git-tracked", "", "track .vine/ in git: true or false (default: false, adds .gitignore)")
	AddJSONFlag(initCmd)
	rootCmd.AddCommand(initCmd)
}
