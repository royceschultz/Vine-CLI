package cmd

import (
	"bufio"
	"fmt"
	"os"
	"path/filepath"
	"strings"

	"github.com/spf13/cobra"

	"vine/config"
)

var configCmd = &cobra.Command{
	Use:   "config",
	Short: "Interactively reconfigure this vine project",
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

		reader := bufio.NewReader(os.Stdin)
		changed := false

		// --- Git tracking ---
		currentGit := "no"
		if cfg.GitTracked {
			currentGit = "yes"
		}
		fmt.Printf("Commit vine tasks to git? (y/n) [%s]: ", currentGit)
		gitChoice, _ := reader.ReadString('\n')
		gitChoice = strings.TrimSpace(strings.ToLower(gitChoice))

		if gitChoice != "" {
			newVal := gitChoice == "y" || gitChoice == "yes"
			if newVal != cfg.GitTracked {
				cfg.GitTracked = newVal
				changed = true
				if cfg.GitTracked {
					gitignorePath := filepath.Join(projectRoot, config.DotVineDir, ".gitignore")
					if err := os.Remove(gitignorePath); err != nil && !os.IsNotExist(err) {
						return fmt.Errorf("removing .gitignore: %w", err)
					}
				} else {
					if err := config.WriteGitIgnore(projectRoot); err != nil {
						return fmt.Errorf("writing .gitignore: %w", err)
					}
				}
			}
		}

		// --- Storage mode (display only for now) ---
		fmt.Printf("Storage mode: %s (not yet reconfigurable)\n", cfg.Storage)

		// --- Project name (display only for now) ---
		fmt.Printf("Project name: %s (not yet reconfigurable)\n", cfg.Name)

		if changed {
			if err := config.Save(projectRoot, cfg); err != nil {
				return err
			}
			fmt.Println("Configuration updated.")
		} else {
			fmt.Println("No changes.")
		}

		return nil
	},
}

var configShowCmd = &cobra.Command{
	Use:   "show",
	Short: "Display current project configuration",
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

		fmt.Printf("Project:     %s\n", cfg.Name)
		fmt.Printf("Storage:     %s\n", cfg.Storage)
		if cfg.Database != "" {
			fmt.Printf("Database:    %s\n", cfg.Database)
		}
		fmt.Printf("Git tracked: %v\n", cfg.GitTracked)

		return nil
	},
}

var configSetCmd = &cobra.Command{
	Use:   "set <key> <value>",
	Short: "Update a configuration value",
	Long:  "Supported keys: git-tracked (true/false)",
	Args:  cobra.ExactArgs(2),
	RunE: func(cmd *cobra.Command, args []string) error {
		key, value := args[0], args[1]

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

		switch key {
		case "git-tracked":
			switch value {
			case "true", "yes":
				cfg.GitTracked = true
				// Remove .gitignore if it exists.
				gitignorePath := filepath.Join(projectRoot, config.DotVineDir, ".gitignore")
				if err := os.Remove(gitignorePath); err != nil && !os.IsNotExist(err) {
					return fmt.Errorf("removing .gitignore: %w", err)
				}
			case "false", "no":
				cfg.GitTracked = false
				if err := config.WriteGitIgnore(projectRoot); err != nil {
					return fmt.Errorf("writing .gitignore: %w", err)
				}
			default:
				return fmt.Errorf("invalid value for git-tracked: %q (use true or false)", value)
			}
		default:
			return fmt.Errorf("unknown config key: %q\nSupported keys: git-tracked", key)
		}

		if err := config.Save(projectRoot, cfg); err != nil {
			return err
		}

		fmt.Printf("Set %s = %s\n", key, value)
		return nil
	},
}

func init() {
	configCmd.AddCommand(configShowCmd)
	configCmd.AddCommand(configSetCmd)
	rootCmd.AddCommand(configCmd)
}
