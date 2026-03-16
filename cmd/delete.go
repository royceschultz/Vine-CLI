package cmd

import (
	"fmt"

	"github.com/spf13/cobra"

	"vine/utils"
)

var deleteCmd = &cobra.Command{
	Use:   "delete <task-id> [task-id...]",
	Short: "Permanently delete tasks (management only — agents should use close/cancel)",
	Long: `Permanently delete tasks from the database.

This is a destructive management operation. Deleted tasks cannot be recovered.
Any subtasks, dependencies, tags, and comments are also removed.

AI agents should NOT use this command. Use "vine close" or "vine cancel" instead,
which preserve task history.`,
	Args: cobra.MinimumNArgs(1),
	RunE: func(cmd *cobra.Command, args []string) error {
		s := GetStore(cmd)
		projectName := getProjectName(cmd)
		force, _ := cmd.Flags().GetBool("force")

		type deleteTarget struct {
			bareID    string
			displayID string
			name      string
			children  int
		}

		var targets []deleteTarget

		for _, arg := range args {
			_, bareID := utils.ParseTaskID(arg)
			task, err := s.GetTask(bareID)
			if err != nil {
				return fmt.Errorf("task %q not found", arg)
			}
			children, _ := s.ChildTasks(bareID)
			targets = append(targets, deleteTarget{
				bareID:    bareID,
				displayID: utils.FormatTaskID(projectName, task.ID),
				name:      task.Name,
				children:  len(children),
			})
		}

		if !force {
			fmt.Println("The following tasks will be permanently deleted:")
			for _, t := range targets {
				suffix := ""
				if t.children > 0 {
					suffix = fmt.Sprintf(" (and %d subtask(s))", t.children)
				}
				fmt.Printf("  %s  %s%s\n", t.displayID, t.name, suffix)
			}
			fmt.Print("\nThis cannot be undone. Re-run with --force to confirm.\n")
			return nil
		}

		var deleted []any
		for _, t := range targets {
			if err := s.DeleteTask(t.bareID); err != nil {
				return err
			}
			if !IsJSON(cmd) {
				suffix := ""
				if t.children > 0 {
					suffix = fmt.Sprintf(" (and %d subtask(s))", t.children)
				}
				fmt.Printf("deleted %s: %s%s\n", t.displayID, t.name, suffix)
			}
			deleted = append(deleted, map[string]string{"id": t.bareID, "name": t.name})
		}

		if IsJSON(cmd) {
			PrintOutput(cmd, "", deleted)
		}

		return nil
	},
}

func init() {
	deleteCmd.Flags().Bool("force", false, "confirm permanent deletion")
	AddJSONFlag(deleteCmd)
	rootCmd.AddCommand(deleteCmd)
}
