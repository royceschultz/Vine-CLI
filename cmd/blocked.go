package cmd

import (
	"fmt"
	"strings"

	"github.com/spf13/cobra"

	"vine/store"
	"vine/utils"
)

var blockedCmd = &cobra.Command{
	Use:   "blocked",
	Short: "List tasks that are blocked by dependencies",
	RunE: func(cmd *cobra.Command, args []string) error {
		var tasks []store.Task
		var err error

		if IsRemote(cmd) {
			c, project := GetRemoteClient(cmd)
			tasks, err = c.BlockedTasks(project)
		} else {
			s := GetStore(cmd)
			tasks, err = s.BlockedTasks()
		}
		if err != nil {
			return err
		}

		if IsRoot(cmd) {
			tasks = filterRoot(tasks)
		}

		// All tasks from BlockedTasks() are blocked by definition.
		for i := range tasks {
			tasks[i].Status = "blocked"
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

		var counts map[string]int
		if !IsRemote(cmd) {
			s := GetStore(cmd)
			counts = collectChildCounts(s, tasks)
		}

		statusColor := utils.StatusColor("blocked")
		statusLabel := statusColor.Sprint("blocked")
		fmt.Printf("%s %s:\n\n", utils.Bold(fmt.Sprintf("%d", len(tasks))), statusLabel)

		indent := "              "
		maxDesc := utils.TermWidth() - len(indent)

		for _, t := range tasks {
			displayID := utils.FormatTaskID(projectName, t.ID)
			fmt.Printf("  %s  %s%s\n", utils.Dim(displayID), utils.Bold(t.Name), utils.TypeLabel(t.Type))

			if t.Description != "" {
				fmt.Printf("%s%s\n", indent, utils.Dim(utils.Truncate(t.Description, maxDesc)))
			}

			if !IsRemote(cmd) {
				s := GetStore(cmd)
				deps, _ := s.DependenciesOf(t.ID)
				var blockers []*store.Task
				for _, d := range deps {
					if dep, err := s.GetTask(d.DependsOnID); err == nil {
						if dep.Status != "done" && dep.Status != "cancelled" {
							blockers = append(blockers, dep)
						}
					}
				}
				if lines := blockerLines(projectName, blockers); lines != "" {
					fmt.Print(lines)
				}
			}

			if subLabel := subtaskLabel(counts, t.ID); subLabel != "" {
				fmt.Printf("%s%s\n", indent, strings.TrimLeft(subLabel, " "))
			}
		}

		return nil
	},
}

func init() {
	AddRootFlag(blockedCmd)
	AddJSONFlag(blockedCmd)
	rootCmd.AddCommand(blockedCmd)
}
