package cmd

import (
	"fmt"

	"github.com/spf13/cobra"

	"vine/store"
	"vine/utils"
)

var subtaskCmd = &cobra.Command{
	Use:   "subtask",
	Short: "Manage sub-task relationships",
}

var subtaskAddCmd = &cobra.Command{
	Use:   "add <parent-id> <child-id>",
	Short: "Set a task as a subtask of another",
	Args:  cobra.ExactArgs(2),
	RunE: func(cmd *cobra.Command, args []string) error {
		s := GetStore(cmd)

		_, parentID := utils.ParseTaskID(args[0])
		_, childID := utils.ParseTaskID(args[1])

		if parentID == childID {
			return fmt.Errorf("a task cannot be its own subtask")
		}

		// Verify both exist.
		parent, err := s.GetTask(parentID)
		if err != nil {
			return fmt.Errorf("task %q not found", args[0])
		}
		child, err := s.GetTask(childID)
		if err != nil {
			return fmt.Errorf("task %q not found", args[1])
		}

		// Prevent circular: child can't already be an ancestor of parent.
		if isAncestor(s, childID, parentID) {
			return fmt.Errorf("circular relationship: %q is already an ancestor of %q", args[1], args[0])
		}

		updated, err := s.SetParent(childID, &parentID)
		if err != nil {
			return err
		}

		projectName := getProjectName(cmd)
		displayParent := utils.FormatTaskID(projectName, parent.ID)
		displayChild := utils.FormatTaskID(projectName, child.ID)

		PrintOutput(cmd, fmt.Sprintf("%s is now a subtask of %s", displayChild, displayParent), updated)
		return nil
	},
}

var subtaskRemoveCmd = &cobra.Command{
	Use:   "remove <child-id>",
	Short: "Remove a task's parent (make it a root task)",
	Args:  cobra.ExactArgs(1),
	RunE: func(cmd *cobra.Command, args []string) error {
		s := GetStore(cmd)

		_, childID := utils.ParseTaskID(args[0])

		task, err := s.GetTask(childID)
		if err != nil {
			return fmt.Errorf("task %q not found", args[0])
		}

		if task.ParentID == nil {
			return fmt.Errorf("task %q has no parent", args[0])
		}

		updated, err := s.SetParent(childID, nil)
		if err != nil {
			return err
		}

		projectName := getProjectName(cmd)
		displayChild := utils.FormatTaskID(projectName, updated.ID)

		PrintOutput(cmd, fmt.Sprintf("%s is now a root task", displayChild), updated)
		return nil
	},
}

var subtaskListCmd = &cobra.Command{
	Use:   "list <parent-id>",
	Short: "List subtasks of a task",
	Args:  cobra.ExactArgs(1),
	RunE: func(cmd *cobra.Command, args []string) error {
		s := GetStore(cmd)

		_, parentID := utils.ParseTaskID(args[0])

		children, err := s.ChildTasks(parentID)
		if err != nil {
			return err
		}

		if IsJSON(cmd) {
			PrintOutput(cmd, "", children)
			return nil
		}

		if len(children) == 0 {
			fmt.Println("No subtasks.")
			return nil
		}

		projectName := getProjectName(cmd)

		for _, c := range children {
			displayID := utils.FormatTaskID(projectName, c.ID)
			statusLabel := utils.StatusColor(c.Status).Sprint(c.Status)
			typeLabel := ""
			if c.Type != "task" {
				typeLabel = " " + utils.Dim("["+c.Type+"]")
			}
			fmt.Printf("  %s  %s  %s%s\n", utils.Dim(displayID), statusLabel, c.Name, typeLabel)
		}

		return nil
	},
}

// isAncestor checks if ancestorID is an ancestor of taskID by walking up the parent chain.
func isAncestor(s *store.Store, ancestorID, taskID string) bool {
	current := taskID
	for i := 0; i < 100; i++ { // depth limit
		task, err := s.GetTask(current)
		if err != nil || task.ParentID == nil {
			return false
		}
		if *task.ParentID == ancestorID {
			return true
		}
		current = *task.ParentID
	}
	return false
}

func init() {
	AddJSONFlag(subtaskAddCmd)
	AddJSONFlag(subtaskRemoveCmd)
	AddJSONFlag(subtaskListCmd)
	subtaskCmd.AddCommand(subtaskAddCmd)
	subtaskCmd.AddCommand(subtaskRemoveCmd)
	subtaskCmd.AddCommand(subtaskListCmd)
	rootCmd.AddCommand(subtaskCmd)
}
