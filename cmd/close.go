package cmd

import (
	"fmt"

	"github.com/spf13/cobra"

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

		return nil
	},
}

func init() {
	closeCmd.Flags().StringP("reason", "r", "", "reason for closing")
	AddJSONFlag(closeCmd)
	rootCmd.AddCommand(closeCmd)
}
