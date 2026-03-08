package cmd

import (
	"context"
	"fmt"
	"os"

	"github.com/spf13/cobra"

	"vine/config"
	"vine/store"
)

type contextKey string

const storeKey contextKey = "store"

var rootCmd = &cobra.Command{
	Use:   "vine",
	Short: "Task tracking for AI agents",
	Long:  "Pick off tasks like grapes on a vine.",
	PersistentPreRunE: func(cmd *cobra.Command, args []string) error {
		// Commands that don't need a database connection.
		if !commandNeedsDB(cmd) {
			return nil
		}

		cwd, err := os.Getwd()
		if err != nil {
			return fmt.Errorf("getting working directory: %w", err)
		}
		projectRoot, err := config.FindProjectRoot(cwd)
		if err != nil {
			return err
		}
		cfg, err := config.Load(projectRoot)
		if err != nil {
			return err
		}
		s, err := store.Open(projectRoot, cfg)
		if err != nil {
			return err
		}
		ctx := context.WithValue(cmd.Context(), storeKey, s)
		cmd.SetContext(ctx)
		return nil
	},
	PersistentPostRunE: func(cmd *cobra.Command, args []string) error {
		if s := storeFromContext(cmd.Context()); s != nil {
			return s.Close()
		}
		return nil
	},
}

func storeFromContext(ctx context.Context) *store.Store {
	if s, ok := ctx.Value(storeKey).(*store.Store); ok {
		return s
	}
	return nil
}

// GetStore retrieves the store from the command context.
func GetStore(cmd *cobra.Command) *store.Store {
	s := storeFromContext(cmd.Context())
	if s == nil {
		fmt.Fprintln(os.Stderr, "error: database not initialized (run 'vine init' first)")
		os.Exit(1)
	}
	return s
}

// commandNeedsDB returns false for commands that should work without a database.
func commandNeedsDB(cmd *cobra.Command) bool {
	// Walk up the command tree to check names.
	for c := cmd; c != nil; c = c.Parent() {
		switch c.Name() {
		case "init", "hello", "help", "completion", "db", "migrate", "onboard", "doctor", "remote":
			return false
		}
	}
	return true
}

func Execute() {
	if err := rootCmd.Execute(); err != nil {
		fmt.Fprintln(os.Stderr, err)
		os.Exit(1)
	}
}
