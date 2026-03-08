package cmd

import (
	"fmt"
	"os"
	"path/filepath"
	"strings"

	"github.com/spf13/cobra"

	"vine/utils"
)

type pruneTarget struct {
	Path string
	Size int64
}

var pruneCmd = &cobra.Command{
	Use:   "prune",
	Short: "Clean temporary and accumulated files",
	Long: `Remove temporary files accumulated by vine (logs, rotated logs, server config).
Does NOT touch database files, WAL files, or SHM files.

Examples:
  vine prune            # delete temp files
  vine prune --dry-run  # show what would be deleted`,
	RunE: func(cmd *cobra.Command, args []string) error {
		dryRun, _ := cmd.Flags().GetBool("dry-run")

		home, err := os.UserHomeDir()
		if err != nil {
			return err
		}
		vineDir := filepath.Join(home, ".vine")

		targets := findPruneTargets(vineDir)

		if len(targets) == 0 {
			fmt.Println("Nothing to clean.")
			return nil
		}

		var totalSize int64
		for _, t := range targets {
			totalSize += t.Size
		}

		if dryRun {
			fmt.Printf("Would delete %d file(s), freeing %s:\n\n", len(targets), formatSize(totalSize))
			for _, t := range targets {
				fmt.Printf("  %s  %s\n", utils.Dim(formatSize(t.Size)), t.Path)
			}
			return nil
		}

		var deleted int
		var freed int64
		for _, t := range targets {
			if err := os.Remove(t.Path); err != nil {
				fmt.Fprintf(os.Stderr, "  warning: could not remove %s: %v\n", t.Path, err)
				continue
			}
			deleted++
			freed += t.Size
		}

		fmt.Printf("Deleted %d file(s), freed %s.\n", deleted, formatSize(freed))
		return nil
	},
}

func findPruneTargets(vineDir string) []pruneTarget {
	var targets []pruneTarget

	// Server log files (server.log, server.log.1, server.log.2, etc.)
	matches, _ := filepath.Glob(filepath.Join(vineDir, "server.log*"))
	for _, m := range matches {
		info, err := os.Stat(m)
		if err != nil || info.IsDir() {
			continue
		}
		targets = append(targets, pruneTarget{Path: m, Size: info.Size()})
	}

	// Server config (server.json) — safe to delete, recreated on next serve.
	serverJSON := filepath.Join(vineDir, "server.json")
	if info, err := os.Stat(serverJSON); err == nil && !info.IsDir() {
		targets = append(targets, pruneTarget{Path: serverJSON, Size: info.Size()})
	}

	// Stale PID file (only if the process isn't running).
	pidFile := filepath.Join(vineDir, "server.pid")
	if info, err := os.Stat(pidFile); err == nil && !info.IsDir() {
		if pid, err := readPID(pidFile); err != nil || !processExists(pid) {
			targets = append(targets, pruneTarget{Path: pidFile, Size: info.Size()})
		}
	}

	return targets
}

func formatSize(bytes int64) string {
	switch {
	case bytes >= 1024*1024*1024:
		return fmt.Sprintf("%.1f GB", float64(bytes)/(1024*1024*1024))
	case bytes >= 1024*1024:
		return fmt.Sprintf("%.1f MB", float64(bytes)/(1024*1024))
	case bytes >= 1024:
		return fmt.Sprintf("%.1f KB", float64(bytes)/1024)
	default:
		return fmt.Sprintf("%d B", bytes)
	}
}

// safeToPrune returns true if the filename is safe to delete.
// This is a guard — we never prune database or WAL/SHM files.
func safeToPrune(name string) bool {
	lower := strings.ToLower(name)
	if strings.HasSuffix(lower, ".db") || strings.HasSuffix(lower, "-wal") || strings.HasSuffix(lower, "-shm") {
		return false
	}
	return true
}

func init() {
	pruneCmd.Flags().Bool("dry-run", false, "show what would be deleted without deleting")
	rootCmd.AddCommand(pruneCmd)
}
