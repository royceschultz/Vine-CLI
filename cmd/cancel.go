package cmd

import (
	"fmt"

	"github.com/spf13/cobra"

	"vine/utils"
)

var cancelCmd = &cobra.Command{
	Use:   "cancel <task-id> [task-id...]",
	Short: "Cancel tasks",
	Args:  cobra.MinimumNArgs(1),
	RunE: func(cmd *cobra.Command, args []string) error {
		s := GetStore(cmd)
		projectName := getProjectName(cmd)
		reason, _ := cmd.Flags().GetString("reason")

		var cancelled []any

		for _, arg := range args {
			_, bareID := utils.ParseTaskID(arg)

			task, err := s.GetTask(bareID)
			if err != nil {
				return fmt.Errorf("task %q not found", arg)
			}

			if task.Status == "cancelled" {
				if !IsJSON(cmd) {
					displayID := utils.FormatTaskID(projectName, task.ID)
					fmt.Printf("%s is already cancelled\n", displayID)
				}
				cancelled = append(cancelled, task)
				continue
			}

			if task.Status == "done" {
				return fmt.Errorf("task %q is already done — cannot cancel", arg)
			}

			updated, err := s.UpdateTaskStatus(bareID, "cancelled")
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
				fmt.Printf("cancelled %s: %s\n", displayID, updated.Name)
			}
			cancelled = append(cancelled, updated)
		}

		if IsJSON(cmd) {
			PrintOutput(cmd, "", cancelled)
		}

		return nil
	},
}

func init() {
	cancelCmd.Flags().StringP("reason", "r", "", "reason for cancelling")
	AddJSONFlag(cancelCmd)
	rootCmd.AddCommand(cancelCmd)
}
