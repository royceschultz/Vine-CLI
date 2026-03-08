package cmd

import (
	"context"
	"fmt"
	"os"

	"github.com/spf13/cobra"

	"vine/client"
	"vine/config"
	"vine/store"
)

type contextKey string

const (
	storeKey         contextKey = "store"
	remoteClientKey  contextKey = "remote_client"
	remoteProjectKey contextKey = "remote_project"
)

var rootCmd = &cobra.Command{
	Use:   "vine",
	Short: "Task tracking for AI agents",
	Long:  "Pick off tasks like grapes on a vine.",
	PersistentPreRunE: func(cmd *cobra.Command, args []string) error {
		// Commands that don't need a database connection.
		if !commandNeedsDB(cmd) {
			return nil
		}

		// Check for --remote mode.
		remoteName, _ := cmd.Root().PersistentFlags().GetString("remote")
		if remoteName != "" {
			return setupRemoteContext(cmd, remoteName)
		}

		// Check for --project mode (direct global database access without .vine/config).
		projectName, _ := cmd.Root().PersistentFlags().GetString("project")
		if projectName != "" {
			return setupProjectContext(cmd, projectName)
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
		// Close SSH tunnel if one was established.
		if c, _ := cmd.Context().Value(remoteClientKey).(*client.Client); c != nil {
			c.Close()
		}
		if s := storeFromContext(cmd.Context()); s != nil {
			return s.Close()
		}
		return nil
	},
}

func init() {
	rootCmd.PersistentFlags().String("remote", "", "query a remote vine server instead of local database")
	rootCmd.PersistentFlags().String("project", "", "query a global database by name (works locally without .vine/config, or specifies project on a remote)")
}

// setupProjectContext opens a global database directly by name.
func setupProjectContext(cmd *cobra.Command, projectName string) error {
	// Warn if there's a .vine/config that we're skipping.
	if cwd, err := os.Getwd(); err == nil {
		if _, err := config.FindProjectRoot(cwd); err == nil {
			fmt.Fprintf(os.Stderr, "note: using --project %q (ignoring local .vine/config)\n", projectName)
		}
	}

	dbDir, err := config.GlobalDatabasesDir()
	if err != nil {
		return err
	}
	dbPath := dbDir + "/" + projectName + ".db"
	if _, err := os.Stat(dbPath); os.IsNotExist(err) {
		return fmt.Errorf("global database %q not found (looked in %s)", projectName, dbDir)
	}
	s, err := store.OpenPath(dbPath)
	if err != nil {
		return err
	}
	ctx := context.WithValue(cmd.Context(), storeKey, s)
	cmd.SetContext(ctx)
	return nil
}

// setupRemoteContext loads the remote config and injects a client into the command context.
func setupRemoteContext(cmd *cobra.Command, remoteName string) error {
	project, _ := cmd.Root().PersistentFlags().GetString("project")
	if project == "" {
		return fmt.Errorf("--project is required when using --remote (specify which project on the remote server)")
	}

	cfg, err := config.LoadRemotes()
	if err != nil {
		return err
	}

	remote := cfg.GetRemote(remoteName)
	if remote == nil {
		return fmt.Errorf("remote %q not found. Use 'vine remote list' to see configured remotes", remoteName)
	}

	c, err := client.New(remote)
	if err != nil {
		return fmt.Errorf("connecting to remote %q: %w", remoteName, err)
	}

	ctx := cmd.Context()
	ctx = context.WithValue(ctx, remoteClientKey, c)
	ctx = context.WithValue(ctx, remoteProjectKey, project)
	cmd.SetContext(ctx)
	return nil
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

// IsRemote returns true if the command is running in remote mode.
func IsRemote(cmd *cobra.Command) bool {
	_, ok := cmd.Context().Value(remoteClientKey).(*client.Client)
	return ok
}

// GetRemoteClient returns the remote client and project name from context.
func GetRemoteClient(cmd *cobra.Command) (*client.Client, string) {
	c, _ := cmd.Context().Value(remoteClientKey).(*client.Client)
	p, _ := cmd.Context().Value(remoteProjectKey).(string)
	return c, p
}

// commandNeedsDB returns false for commands that should work without a database.
func commandNeedsDB(cmd *cobra.Command) bool {
	// Walk up the command tree to check names.
	for c := cmd; c != nil; c = c.Parent() {
		switch c.Name() {
		case "init", "hello", "help", "completion", "db", "migrate", "onboard", "doctor", "remote", "prune", "config":
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
