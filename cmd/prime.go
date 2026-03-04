package cmd

import (
	"fmt"
	"strings"

	"github.com/spf13/cobra"

	"vine/store"
	"vine/utils"
)

var primeCmd = &cobra.Command{
	Use:   "prime",
	Short: "Output context for an AI agent's current session",
	Long: `Outputs a concise workflow context block for AI agents.
Includes project status, ready tasks, and command reference.
Run this at the start of a session or after context compaction.`,
	RunE: func(cmd *cobra.Command, args []string) error {
		s := GetStore(cmd)
		projectName := getProjectName(cmd)

		if IsJSON(cmd) {
			return primeJSON(cmd, s, projectName)
		}
		return primeText(s, projectName)
	},
}

func primeText(s *store.Store, project string) error {
	var b strings.Builder

	b.WriteString("# Vine Workflow Context\n\n")

	// Project status summary.
	counts, _ := s.TaskSummary()
	total := 0
	statusLine := []string{}
	for _, c := range counts {
		total += c.Count
		statusLine = append(statusLine, fmt.Sprintf("%s: %d", c.Status, c.Count))
	}
	b.WriteString(fmt.Sprintf("**%s** — %d tasks (%s)\n\n", project, total, strings.Join(statusLine, ", ")))

	// Ready tasks.
	ready, _ := s.ReadyTasks()
	if len(ready) > 0 {
		b.WriteString("## Ready to work on\n\n")
		limit := len(ready)
		if limit > 10 {
			limit = 10
		}
		for i := 0; i < limit; i++ {
			t := ready[i]
			id := utils.FormatTaskID(project, t.ID)
			typeLabel := ""
			if t.Type != "task" {
				typeLabel = fmt.Sprintf(" [%s]", t.Type)
			}
			desc := ""
			if t.Description != "" {
				desc = " — " + t.Description
			}
			b.WriteString(fmt.Sprintf("- `%s` %s%s%s\n", id, t.Name, typeLabel, desc))
		}
		if len(ready) > limit {
			b.WriteString(fmt.Sprintf("\n*… and %d more — run `vine ready --all` to see all*\n", len(ready)-limit))
		}
		b.WriteString("\n")
	}

	// In-progress tasks.
	inProgress, _ := s.ListTasks("in_progress")
	if len(inProgress) > 0 {
		b.WriteString("## In progress\n\n")
		for _, t := range inProgress {
			id := utils.FormatTaskID(project, t.ID)
			typeLabel := ""
			if t.Type != "task" {
				typeLabel = fmt.Sprintf(" [%s]", t.Type)
			}
			b.WriteString(fmt.Sprintf("- `%s` %s%s\n", id, t.Name, typeLabel))
		}
		b.WriteString("\n")
	}

	// Command reference.
	b.WriteString("## Commands\n\n")
	b.WriteString("- `vine ready` — find tasks ready to work on\n")
	b.WriteString("- `vine pick <id>` — claim a task (sets to in_progress)\n")
	b.WriteString("- `vine show <id>` — view task details, deps, subtasks\n")
	b.WriteString("- `vine create \"Title\" -d \"description\" -t type` — create a task\n")
	b.WriteString("- `vine list -s status -t type --tag name` — filtered list\n")
	b.WriteString("- `vine dep add <task> <depends-on>` — add dependency\n")
	b.WriteString("- `vine subtask add <parent> <child>` — add subtask\n")
	b.WriteString("- `vine status` — project summary\n")
	b.WriteString("\nAll commands support `--json` for structured output. Use `--help` on any command.\n")

	fmt.Print(b.String())
	return nil
}

func primeJSON(cmd *cobra.Command, s *store.Store, project string) error {
	counts, _ := s.TaskSummary()
	ready, _ := s.ReadyTasks()
	inProgress, _ := s.ListTasks("in_progress")

	data := map[string]any{
		"project":     project,
		"status":      counts,
		"ready":       ready,
		"in_progress": inProgress,
	}

	PrintOutput(cmd, "", data)
	return nil
}

func init() {
	AddJSONFlag(primeCmd)
	rootCmd.AddCommand(primeCmd)
}
