package cmd

import (
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"

	"github.com/spf13/cobra"

	"vine/config"
	"vine/store"
	"vine/utils"
)

type checkResult struct {
	Name   string `json:"name"`
	Status string `json:"status"` // "ok", "warn", "error"
	Detail string `json:"detail"`
	Fix    string `json:"fix,omitempty"`
}

var doctorCmd = &cobra.Command{
	Use:   "doctor",
	Short: "Diagnose vine configuration and integration issues",
	RunE: func(cmd *cobra.Command, args []string) error {
		var checks []checkResult
		fix, _ := cmd.Flags().GetBool("fix")

		cwd, err := os.Getwd()
		if err != nil {
			return err
		}

		// 1. Vine project config.
		checks = append(checks, checkVineProject(cwd)...)

		// 2. Database.
		checks = append(checks, checkDatabase(cwd)...)

		// 3. Database symlinks (global storage only).
		checks = append(checks, checkSymlinks(cwd, fix)...)

		// 4. Claude Code local settings (system prompt).
		checks = append(checks, checkClaudeLocalSettings(cwd))

		// 5. Claude Code hooks (local or global settings).
		checks = append(checks, checkClaudeHooks(cwd))

		if IsJSON(cmd) {
			PrintOutput(cmd, "", checks)
			return nil
		}

		hasIssues := false
		for _, c := range checks {
			var icon string
			switch c.Status {
			case "ok":
				icon = utils.StatusColor("ready").Sprint("OK")
			case "warn":
				icon = utils.StatusColor("blocked").Sprint("WARN")
				hasIssues = true
			case "error":
				icon = utils.StatusColor("cancelled").Sprint("ERR")
				hasIssues = true
			}

			fmt.Printf("  [%s]  %s\n", icon, c.Name)
			if c.Detail != "" {
				fmt.Printf("         %s\n", c.Detail)
			}
			if c.Fix != "" {
				fmt.Printf("         %s %s\n", utils.Dim("fix:"), c.Fix)
			}
		}

		if !hasIssues {
			fmt.Printf("\n  All checks passed.\n")
		}

		return nil
	},
}

func checkVineProject(cwd string) []checkResult {
	projectRoot, err := config.FindProjectRoot(cwd)
	if err != nil {
		return []checkResult{{
			Name:   "vine project",
			Status: "error",
			Detail: "No .vine/ directory found in this directory or any parent.",
			Fix:    "Run 'vine init' to set up a new project.",
		}}
	}

	cfg, err := config.Load(projectRoot)
	if err != nil {
		return []checkResult{{
			Name:   "vine project",
			Status: "error",
			Detail: fmt.Sprintf("Found .vine/ at %s but config is invalid: %v", projectRoot, err),
			Fix:    "Check .vine/config.json for syntax errors, or re-run 'vine init'.",
		}}
	}

	name := config.ProjectName(projectRoot, cfg)
	return []checkResult{{
		Name:   "vine project",
		Status: "ok",
		Detail: fmt.Sprintf("%s (storage: %s, root: %s)", name, cfg.Storage, projectRoot),
	}}
}

func checkDatabase(cwd string) []checkResult {
	projectRoot, err := config.FindProjectRoot(cwd)
	if err != nil {
		return nil // already reported by project check
	}

	cfg, err := config.Load(projectRoot)
	if err != nil {
		return nil
	}

	dbPath, err := config.DatabasePath(projectRoot, cfg)
	if err != nil {
		return []checkResult{{
			Name:   "database",
			Status: "error",
			Detail: fmt.Sprintf("Cannot resolve database path: %v", err),
		}}
	}

	if _, err := os.Stat(dbPath); os.IsNotExist(err) {
		return []checkResult{{
			Name:   "database",
			Status: "error",
			Detail: fmt.Sprintf("Database file not found: %s", dbPath),
			Fix:    "Run 'vine init' to create the database.",
		}}
	}

	// Try opening to verify it's valid.
	s, err := store.OpenPath(dbPath)
	if err != nil {
		return []checkResult{{
			Name:   "database",
			Status: "error",
			Detail: fmt.Sprintf("Cannot open database: %v", err),
		}}
	}
	s.Close()

	return []checkResult{{
		Name:   "database",
		Status: "ok",
		Detail: dbPath,
	}}
}

func checkClaudeLocalSettings(cwd string) checkResult {
	settingsPath := filepath.Join(cwd, ".claude", "settings.local.json")

	data, err := os.ReadFile(settingsPath)
	if err != nil {
		if os.IsNotExist(err) {
			return checkResult{
				Name:   "claude: system prompt",
				Status: "warn",
				Detail: "No .claude/settings.local.json found.",
				Fix:    "Run 'vine init claude' to add the vine system prompt.",
			}
		}
		return checkResult{
			Name:   "claude: system prompt",
			Status: "error",
			Detail: fmt.Sprintf("Cannot read %s: %v", settingsPath, err),
		}
	}

	var settings map[string]any
	if err := json.Unmarshal(data, &settings); err != nil {
		return checkResult{
			Name:   "claude: system prompt",
			Status: "error",
			Detail: fmt.Sprintf("Cannot parse %s: %v", settingsPath, err),
		}
	}

	prompt, _ := settings["systemPrompt"].(string)
	if prompt == "" {
		return checkResult{
			Name:   "claude: system prompt",
			Status: "warn",
			Detail: "settings.local.json exists but has no systemPrompt.",
			Fix:    "Run 'vine init claude' to add the vine system prompt.",
		}
	}

	if prompt != vineClaudePrompt {
		return checkResult{
			Name:   "claude: system prompt",
			Status: "ok",
			Detail: "Custom system prompt detected (not the vine default, but present).",
		}
	}

	return checkResult{
		Name:   "claude: system prompt",
		Status: "ok",
		Detail: "Vine system prompt installed.",
	}
}

func checkClaudeHooks(cwd string) checkResult {
	// Check both local (.claude/settings.local.json) and global (~/.claude/settings.json).
	// Hooks in either location are valid.
	var hasSession, hasCompact bool

	for _, settingsPath := range claudeHookPaths(cwd) {
		data, err := os.ReadFile(settingsPath)
		if err != nil {
			continue
		}
		var settings map[string]any
		if err := json.Unmarshal(data, &settings); err != nil {
			continue
		}
		hooks, _ := settings["hooks"].(map[string]any)
		if hooks == nil {
			continue
		}
		if hasVineHook(hooks, "SessionStart") {
			hasSession = true
		}
		if hasVineHook(hooks, "PreCompact") {
			hasCompact = true
		}
	}

	if hasSession && hasCompact {
		return checkResult{
			Name:   "claude: session hooks",
			Status: "ok",
			Detail: "SessionStart and PreCompact hooks installed.",
		}
	}

	missing := []string{}
	if !hasSession {
		missing = append(missing, "SessionStart")
	}
	if !hasCompact {
		missing = append(missing, "PreCompact")
	}

	return checkResult{
		Name:   "claude: session hooks",
		Status: "warn",
		Detail: fmt.Sprintf("Missing hooks: %v. Vine prime won't run automatically.", missing),
		Fix:    "Run 'vine init claude --hooks' to add hooks to .claude/settings.local.json.",
	}
}

// claudeHookPaths returns the settings files to check for hooks (local first, then global).
func claudeHookPaths(cwd string) []string {
	paths := []string{
		filepath.Join(cwd, ".claude", "settings.local.json"),
	}
	if home, err := os.UserHomeDir(); err == nil {
		paths = append(paths, filepath.Join(home, ".claude", "settings.json"))
	}
	return paths
}

func checkSymlinks(cwd string, fix bool) []checkResult {
	projectRoot, err := config.FindProjectRoot(cwd)
	if err != nil {
		return nil // already reported by project check
	}

	cfg, err := config.Load(projectRoot)
	if err != nil {
		return nil
	}

	if cfg.Storage != config.StorageGlobal {
		return nil // no symlinks needed for local storage
	}

	ok, detail := checkDBSymlinks(projectRoot, cfg)
	if ok {
		return []checkResult{{
			Name:   "database symlinks",
			Status: "ok",
			Detail: detail,
		}}
	}

	if fix {
		if err := createDBSymlinks(projectRoot, cfg); err != nil {
			return []checkResult{{
				Name:   "database symlinks",
				Status: "error",
				Detail: fmt.Sprintf("Failed to fix: %v", err),
			}}
		}
		return []checkResult{{
			Name:   "database symlinks",
			Status: "ok",
			Detail: "Fixed: symlinks recreated.",
		}}
	}

	return []checkResult{{
		Name:   "database symlinks",
		Status: "warn",
		Detail: detail,
		Fix:    "Run 'vine doctor --fix' or 'vine symlink create'.",
	}}
}

func init() {
	doctorCmd.Flags().Bool("fix", false, "automatically fix issues where possible")
	AddJSONFlag(doctorCmd)
	rootCmd.AddCommand(doctorCmd)
}
