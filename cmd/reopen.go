package cmd

import (
	"fmt"

	"github.com/spf13/cobra"

	"vine/utils"
)

var reopenCmd = &cobra.Command{
	Use:   "reopen <task-id> [task-id...]",
	Short: "Reopen closed or cancelled tasks",
	Args:  cobra.MinimumNArgs(1),
	RunE: func(cmd *cobra.Command, args []string) error {
		s := GetStore(cmd)
		projectName := getProjectName(cmd)
		reason, _ := cmd.Flags().GetString("reason")

		var reopened []any

		for _, arg := range args {
			_, bareID := utils.ParseTaskID(arg)

			task, err := s.GetTask(bareID)
			if err != nil {
				return fmt.Errorf("task %q not found", arg)
			}

			if task.Status == "open" || task.Status == "in_progress" {
				if !IsJSON(cmd) {
					displayID := utils.FormatTaskID(projectName, task.ID)
					fmt.Printf("%s is already %s\n", displayID, task.Status)
				}
				reopened = append(reopened, task)
				continue
			}

			updated, err := s.UpdateTaskStatus(bareID, "open")
			if err != nil {
				return err
			}

			if reason != "" {
				if _, err := s.AddComment(bareID, "reopen", reason); err != nil {
					return err
				}
			}

			if !IsJSON(cmd) {
				displayID := utils.FormatTaskID(projectName, updated.ID)
				fmt.Printf("reopened %s: %s\n", displayID, updated.Name)
			}
			reopened = append(reopened, updated)
		}

		if IsJSON(cmd) {
			PrintOutput(cmd, "", reopened)
		}

		return nil
	},
}

func init() {
	reopenCmd.Flags().StringP("reason", "r", "", "reason for reopening")
	AddJSONFlag(reopenCmd)
	rootCmd.AddCommand(reopenCmd)
}
