package cmd

import (
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"strings"

	"github.com/spf13/cobra"
)

const vineClaudePrompt = "Before starting any work, run 'vine onboard' to understand how this project tracks tasks."

// vineHookCommand runs vine prime. Since hooks are installed locally per-project
// (in .claude/settings.local.json), vine is guaranteed to be set up here.
const vineHookCommand = "vine prime"

var initClaudeCmd = &cobra.Command{
	Use:   "claude",
	Short: "Set up vine integration for Claude Code",
	RunE: func(cmd *cobra.Command, args []string) error {
		installHooks, _ := cmd.Flags().GetBool("hooks")

		cwd, err := os.Getwd()
		if err != nil {
			return err
		}

		settingsDir := filepath.Join(cwd, ".claude")
		settingsPath := filepath.Join(settingsDir, "settings.local.json")

		settings, err := readOrCreateSettings(settingsDir, settingsPath)
		if err != nil {
			return err
		}

		changed := false

		// Add system prompt.
		if existing, ok := settings["systemPrompt"].(string); !ok || existing != vineClaudePrompt {
			settings["systemPrompt"] = vineClaudePrompt
			changed = true
			fmt.Println("Added system prompt.")
		}

		// Optionally add hooks.
		if installHooks {
			if addClaudeHooks(settings) {
				changed = true
				fmt.Println("Added PreCompact and SessionStart hooks.")
			}
		}

		if !changed {
			fmt.Println("Claude Code integration already configured.")
			return nil
		}

		if err := writeSettings(settingsPath, settings); err != nil {
			return err
		}

		fmt.Printf("Updated %s\n", settingsPath)
		return nil
	},
}

func readOrCreateSettings(dir, path string) (map[string]any, error) {
	data, err := os.ReadFile(path)
	if err == nil {
		var settings map[string]any
		if err := json.Unmarshal(data, &settings); err != nil {
			return nil, fmt.Errorf("parsing %s: %w", path, err)
		}
		return settings, nil
	}
	if os.IsNotExist(err) {
		if err := os.MkdirAll(dir, 0o755); err != nil {
			return nil, err
		}
		return make(map[string]any), nil
	}
	return nil, err
}

func writeSettings(path string, settings map[string]any) error {
	out, err := json.MarshalIndent(settings, "", "  ")
	if err != nil {
		return fmt.Errorf("marshaling settings: %w", err)
	}
	return os.WriteFile(path, append(out, '\n'), 0o644)
}

// addClaudeHooks adds PreCompact and SessionStart hooks if not already present.
// Returns true if hooks were modified.
func addClaudeHooks(settings map[string]any) bool {
	hookEntry := map[string]any{
		"matcher": "",
		"hooks": []any{
			map[string]any{
				"type":    "command",
				"command": vineHookCommand,
			},
		},
	}

	hooks, _ := settings["hooks"].(map[string]any)
	if hooks == nil {
		hooks = make(map[string]any)
	}

	changed := false
	for _, event := range []string{"PreCompact", "SessionStart"} {
		if !hasVineHook(hooks, event) {
			existing, _ := hooks[event].([]any)
			hooks[event] = append(existing, hookEntry)
			changed = true
		}
	}

	if changed {
		settings["hooks"] = hooks
	}
	return changed
}

// hasVineHook checks if a vine hook already exists for the given event.
// Matches any hook command containing "vine prime" (covers both the safe-wrapped
// version and direct "vine prime").
func hasVineHook(hooks map[string]any, event string) bool {
	entries, ok := hooks[event].([]any)
	if !ok {
		return false
	}
	for _, entry := range entries {
		e, ok := entry.(map[string]any)
		if !ok {
			continue
		}
		hookList, ok := e["hooks"].([]any)
		if !ok {
			continue
		}
		for _, h := range hookList {
			hook, ok := h.(map[string]any)
			if !ok {
				continue
			}
			if cmd, ok := hook["command"].(string); ok && strings.Contains(cmd, "vine prime") {
				return true
			}
		}
	}
	return false
}

var initCursorCmd = &cobra.Command{
	Use:   "cursor",
	Short: "Set up vine integration for Cursor",
	Run: func(cmd *cobra.Command, args []string) {
		fmt.Fprintln(os.Stderr, "not yet implemented")
		os.Exit(1)
	},
}

var initCopilotCmd = &cobra.Command{
	Use:   "copilot",
	Short: "Set up vine integration for GitHub Copilot",
	Run: func(cmd *cobra.Command, args []string) {
		fmt.Fprintln(os.Stderr, "not yet implemented")
		os.Exit(1)
	},
}

func init() {
	initClaudeCmd.Flags().Bool("hooks", false, "install PreCompact and SessionStart hooks in .claude/settings.local.json")
	initCmd.AddCommand(initClaudeCmd)
	initCmd.AddCommand(initCursorCmd)
	initCmd.AddCommand(initCopilotCmd)
}
