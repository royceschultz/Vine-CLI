package cmd

import (
	"fmt"
	"strings"

	"github.com/spf13/cobra"

	"vine/store"
	"vine/utils"
)

var updateCmd = &cobra.Command{
	Use:   "update <task-id>",
	Short: "Update task fields",
	Args:  cobra.ExactArgs(1),
	RunE: func(cmd *cobra.Command, args []string) error {
		s := GetStore(cmd)

		_, bareID := utils.ParseTaskID(args[0])

		if _, err := s.GetTask(bareID); err != nil {
			return fmt.Errorf("task %q not found", args[0])
		}

		params := store.UpdateTaskParams{}
		changed := false

		if cmd.Flags().Changed("name") {
			v, _ := cmd.Flags().GetString("name")
			if strings.TrimSpace(v) == "" {
				return fmt.Errorf("--name cannot be empty")
			}
			params.Name = &v
			changed = true
		}
		if cmd.Flags().Changed("description") {
			v, _ := cmd.Flags().GetString("description")
			params.Description = &v
			changed = true
		}
		if cmd.Flags().Changed("details") {
			v, _ := cmd.Flags().GetString("details")
			params.Details = &v
			changed = true
		}
		if cmd.Flags().Changed("type") {
			v, _ := cmd.Flags().GetString("type")
			params.Type = &v
			changed = true
		}

		// Handle tag additions/removals.
		addTags, _ := cmd.Flags().GetStringSlice("add-tag")
		rmTags, _ := cmd.Flags().GetStringSlice("rm-tag")

		if !changed && len(addTags) == 0 && len(rmTags) == 0 {
			return fmt.Errorf("no fields to update (use --name, --description, --details, --type, --add-tag, or --rm-tag)")
		}

		updated, err := s.UpdateTask(bareID, params)
		if err != nil {
			return err
		}

		for _, tag := range addTags {
			if err := s.AddTag(bareID, tag); err != nil {
				return err
			}
		}
		for _, tag := range rmTags {
			if err := s.RemoveTag(bareID, tag); err != nil {
				return fmt.Errorf("removing tag %q: %w", tag, err)
			}
		}

		projectName := getProjectName(cmd)
		displayID := utils.FormatTaskID(projectName, updated.ID)
		PrintOutput(cmd, fmt.Sprintf("updated %s: %s", displayID, updated.Name), updated)

		return nil
	},
}

func init() {
	updateCmd.Flags().String("name", "", "new task name")
	updateCmd.Flags().StringP("description", "d", "", "new description")
	updateCmd.Flags().String("details", "", "new details")
	updateCmd.Flags().StringP("type", "t", "", "new type (feature, bug, task, epic)")
	updateCmd.Flags().StringSlice("add-tag", nil, "tags to add")
	updateCmd.Flags().StringSlice("rm-tag", nil, "tags to remove")
	AddJSONFlag(updateCmd)
	rootCmd.AddCommand(updateCmd)
}
