package cmd

import (
	"fmt"

	"github.com/spf13/cobra"

	"vine/store"
	"vine/utils"
)

var pickCmd = &cobra.Command{
	Use:   "pick <task-id>",
	Short: "Mark a task as in-progress",
	Args:  cobra.ExactArgs(1),
	RunE: func(cmd *cobra.Command, args []string) error {
		s := GetStore(cmd)

		_, bareID := utils.ParseTaskID(args[0])

		task, err := s.GetTask(bareID)
		if err != nil {
			return fmt.Errorf("task %q not found", args[0])
		}

		if task.Status == "in_progress" {
			projectName := getProjectName(cmd)
			displayID := utils.FormatTaskID(projectName, task.ID)
			PrintOutput(cmd, fmt.Sprintf("%s is already in progress", displayID), task)
			return nil
		}

		if task.Status != "open" {
			return fmt.Errorf("cannot pick task with status %q (must be open)", task.Status)
		}

		updated, err := s.UpdateTaskStatus(bareID, "in_progress")
		if err != nil {
			return err
		}

		// Update metadata with current branch/dir.
		meta := detectMetadata()
		existing, _ := updated.ParseMetadata()
		if existing != nil {
			meta.CreatedBranch = existing.CreatedBranch
			meta.CreatedDir = existing.CreatedDir
			meta.CreatedBy = existing.CreatedBy
		}
		updated, err = s.UpdateTaskMetadata(bareID, &store.TaskMetadata{
			CreatedBranch: meta.CreatedBranch,
			CreatedDir:    meta.CreatedDir,
			CreatedBy:     meta.CreatedBy,
			UpdatedBranch: meta.UpdatedBranch,
			UpdatedDir:    meta.UpdatedDir,
			UpdatedBy:     meta.UpdatedBy,
		})
		if err != nil {
			return err
		}

		projectName := getProjectName(cmd)
		displayID := utils.FormatTaskID(projectName, updated.ID)
		PrintOutput(cmd, fmt.Sprintf("picked %s: %s", displayID, updated.Name), updated)

		return nil
	},
}

func init() {
	AddJSONFlag(pickCmd)
	rootCmd.AddCommand(pickCmd)
}
