package cmd

import (
	"fmt"

	"github.com/spf13/cobra"

	"vine/store"
	"vine/utils"
)

var blockedCmd = &cobra.Command{
	Use:   "blocked",
	Short: "List tasks that are blocked by dependencies",
	RunE: func(cmd *cobra.Command, args []string) error {
		s := GetStore(cmd)

		tasks, err := s.BlockedTasks()
		if err != nil {
			return err
		}

		if IsRoot(cmd) {
			tasks = filterRoot(tasks)
		}

		if IsJSON(cmd) {
			PrintOutput(cmd, "", tasks)
			return nil
		}

		if len(tasks) == 0 {
			fmt.Println("No blocked tasks.")
			return nil
		}

		projectName := getProjectName(cmd)
		counts := collectChildCounts(s, tasks)

		fmt.Printf("%s blocked:\n\n", utils.Bold(fmt.Sprintf("%d", len(tasks))))
		for _, t := range tasks {
			displayID := utils.FormatTaskID(projectName, t.ID)
			typeLabel := ""
			if t.Type != "task" {
				typeLabel = " " + utils.Dim("["+t.Type+"]")
			}

			// Show what's blocking this task.
			deps, _ := s.DependenciesOf(t.ID)
			var blockers []*store.Task
			for _, d := range deps {
				if dep, err := s.GetTask(d.DependsOnID); err == nil {
					if dep.Status != "done" && dep.Status != "cancelled" {
						blockers = append(blockers, dep)
					}
				}
			}

			bLabel := blockerLabel(projectName, blockers)

			subLabel := subtaskLabel(counts, t.ID)

			fmt.Printf("  %s  %s%s%s%s\n", utils.Dim(displayID), t.Name, typeLabel, bLabel, subLabel)
		}

		return nil
	},
}

func init() {
	AddRootFlag(blockedCmd)
	AddJSONFlag(blockedCmd)
	rootCmd.AddCommand(blockedCmd)
}
