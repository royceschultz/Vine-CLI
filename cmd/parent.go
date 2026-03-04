package cmd

import (
	"fmt"

	"github.com/spf13/cobra"

	"vine/utils"
)

var parentCmd = &cobra.Command{
	Use:   "parent <task-id>",
	Short: "Show the parent and ancestor chain of a task",
	Args:  cobra.ExactArgs(1),
	RunE: func(cmd *cobra.Command, args []string) error {
		s := GetStore(cmd)
		projectName := getProjectName(cmd)

		_, bareID := utils.ParseTaskID(args[0])

		task, err := s.GetTask(bareID)
		if err != nil {
			return fmt.Errorf("task %q not found", args[0])
		}

		if task.ParentID == nil {
			if IsJSON(cmd) {
				PrintOutput(cmd, "", []any{})
				return nil
			}
			fmt.Println("No parent (root task).")
			return nil
		}

		ancestors, err := s.AncestorChain(bareID)
		if err != nil {
			return err
		}

		if IsJSON(cmd) {
			PrintOutput(cmd, "", ancestors)
			return nil
		}

		for i, a := range ancestors {
			aDisplay := utils.FormatTaskID(projectName, a.ID)
			statusLabel := utils.StatusColor(a.Status).Sprint(a.Status)
			depth := ""
			if i == 0 {
				depth = "parent:      "
			} else if i == len(ancestors)-1 {
				depth = "root:        "
			} else {
				depth = "grandparent: "
			}
			fmt.Printf("  %s%s  %s  %s\n", utils.Dim(depth), utils.Dim(aDisplay), statusLabel, a.Name)
		}

		return nil
	},
}

func init() {
	AddJSONFlag(parentCmd)
	rootCmd.AddCommand(parentCmd)
}
