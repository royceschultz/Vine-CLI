package cmd

import (
	"fmt"
	"strings"

	"github.com/spf13/cobra"

	"vine/store"
	"vine/utils"
)

var closeCmd = &cobra.Command{
	Use:   "close <task-id> [task-id...]",
	Short: "Mark tasks as done",
	Args:  cobra.MinimumNArgs(1),
	RunE: func(cmd *cobra.Command, args []string) error {
		s := GetStore(cmd)
		projectName := getProjectName(cmd)
		reason, _ := cmd.Flags().GetString("reason")

		var closed []any

		for _, arg := range args {
			_, bareID := utils.ParseTaskID(arg)

			task, err := s.GetTask(bareID)
			if err != nil {
				return fmt.Errorf("task %q not found", arg)
			}

			if task.Status == "done" {
				if !IsJSON(cmd) {
					displayID := utils.FormatTaskID(projectName, task.ID)
					fmt.Printf("%s is already done\n", displayID)
				}
				closed = append(closed, task)
				continue
			}

			if task.Status == "cancelled" {
				return fmt.Errorf("task %q is cancelled — reopen it first", arg)
			}

			incomplete, err := s.IncompleteChildTasks(bareID)
			if err != nil {
				return err
			}
			if len(incomplete) > 0 {
				lines := make([]string, len(incomplete))
				for i, c := range incomplete {
					lines[i] = fmt.Sprintf("  %s %s", utils.FormatTaskID(projectName, c.ID), c.Name)
				}
				return fmt.Errorf("cannot close %s — %d incomplete subtask(s):\n%s",
					utils.FormatTaskID(projectName, task.ID), len(incomplete), strings.Join(lines, "\n"))
			}

			updated, err := s.UpdateTaskStatus(bareID, "done")
			if err != nil {
				return err
			}

			if reason != "" {
				if _, err := s.AddComment(bareID, "close", reason); err != nil {
					return err
				}
			}

			if !IsJSON(cmd) {
				displayID := utils.FormatTaskID(projectName, updated.ID)
				fmt.Printf("closed %s: %s\n", displayID, updated.Name)
			}
			closed = append(closed, updated)
		}

		if IsJSON(cmd) {
			PrintOutput(cmd, "", closed)
		}

		// Hint if a parent task's subtasks are now all complete.
		if !IsJSON(cmd) {
			hintedParents := map[string]bool{}
			for _, c := range closed {
				task, ok := c.(*store.Task)
				if !ok || task.ParentID == nil {
					continue
				}
				parentID := *task.ParentID
				if hintedParents[parentID] {
					continue
				}
				hintedParents[parentID] = true
				parent, err := s.GetTask(parentID)
				if err != nil || parent.Status == "done" || parent.Status == "cancelled" {
					continue
				}
				incomplete, err := s.IncompleteChildTasks(parentID)
				if err != nil || len(incomplete) > 0 {
					continue
				}
				parentDisplay := utils.FormatTaskID(projectName, parentID)
				fmt.Printf("\nhint: all subtasks of %s are done — run vine close %s to close it\n", parentDisplay, parentDisplay)
			}
		}

		return nil
	},
}

func init() {
	closeCmd.Flags().StringP("reason", "r", "", "reason for closing")
	AddJSONFlag(closeCmd)
	rootCmd.AddCommand(closeCmd)
}
