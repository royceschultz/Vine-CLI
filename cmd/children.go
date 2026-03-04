package cmd

import (
	"fmt"

	"github.com/spf13/cobra"

	"vine/utils"
)

var childrenCmd = &cobra.Command{
	Use:   "children <task-id>",
	Short: "List subtasks of a task",
	Args:  cobra.ExactArgs(1),
	RunE: func(cmd *cobra.Command, args []string) error {
		s := GetStore(cmd)
		projectName := getProjectName(cmd)

		_, bareID := utils.ParseTaskID(args[0])

		parent, err := s.GetTask(bareID)
		if err != nil {
			return fmt.Errorf("task %q not found", args[0])
		}

		children, err := s.ChildTasks(bareID)
		if err != nil {
			return err
		}

		if IsJSON(cmd) {
			PrintOutput(cmd, "", map[string]any{
				"parent":   parent,
				"children": children,
			})
			return nil
		}

		if len(children) == 0 {
			fmt.Println("No subtasks.")
			return nil
		}

		parentDisplay := utils.FormatTaskID(projectName, parent.ID)
		fmt.Printf("%s  %s\n\n", utils.Dim(parentDisplay), parent.Name)

		// Get child counts so we can show which children have their own subtasks.
		childIDs := make([]string, len(children))
		for i, c := range children {
			childIDs[i] = c.ID
		}
		counts, _ := s.ChildCounts(childIDs)

		for _, c := range children {
			childDisplay := utils.FormatTaskID(projectName, c.ID)
			statusLabel := utils.StatusColor(c.Status).Sprint(c.Status)
			typeLabel := ""
			if c.Type != "task" {
				typeLabel = " " + utils.Dim("["+c.Type+"]")
			}
			subLabel := ""
			if n, ok := counts[c.ID]; ok && n > 0 {
				subLabel = " " + utils.Dim(fmt.Sprintf("(%d subtasks)", n))
			}
			fmt.Printf("  %s  %s  %s%s%s\n", utils.Dim(childDisplay), statusLabel, c.Name, typeLabel, subLabel)
		}

		return nil
	},
}

func init() {
	AddJSONFlag(childrenCmd)
	rootCmd.AddCommand(childrenCmd)
}
