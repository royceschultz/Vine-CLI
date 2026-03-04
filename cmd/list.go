package cmd

import (
	"fmt"

	"github.com/spf13/cobra"

	"vine/store"
	"vine/utils"
)

var listCmd = &cobra.Command{
	Use:   "list",
	Short: "Show all tasks",
	Long:  "Show all tasks. Done and cancelled tasks are hidden by default.",
	RunE: func(cmd *cobra.Command, args []string) error {
		s := GetStore(cmd)

		status, _ := cmd.Flags().GetString("status")
		taskType, _ := cmd.Flags().GetString("type")
		tag, _ := cmd.Flags().GetString("tag")
		showAll, _ := cmd.Flags().GetBool("all")
		n, _ := cmd.Flags().GetInt("number")

		tasks, err := s.ListTasksFiltered(store.TaskFilter{
			Status:   status,
			Type:     taskType,
			Tag:      tag,
			All:      showAll,
			RootOnly: IsRoot(cmd),
		})
		if err != nil {
			return err
		}

		if IsJSON(cmd) {
			PrintOutput(cmd, "", tasks)
			return nil
		}

		if len(tasks) == 0 {
			fmt.Println("No tasks found.")
			return nil
		}

		projectName := getProjectName(cmd)
		counts := collectChildCounts(s, tasks)
		parents := collectParents(s, tasks)

		limit := len(tasks)
		if n > 0 && n < limit {
			limit = n
		}

		for i := 0; i < limit; i++ {
			t := tasks[i]
			displayID := utils.FormatTaskID(projectName, t.ID)

			statusLabel := utils.StatusColor(t.Status).Sprintf("%-12s", t.Status)
			typeLabel := ""
			if t.Type != "task" {
				typeLabel = " " + utils.Dim("["+t.Type+"]")
			}

			pLabel := parentLabel(projectName, parents, t.ParentID)
			subLabel := subtaskLabel(counts, t.ID)

			fmt.Printf("  %s  %s  %s%s%s%s\n", utils.Dim(displayID), statusLabel, t.Name, typeLabel, pLabel, subLabel)
		}

		if limit < len(tasks) {
			fmt.Printf("\n  %s\n", utils.Dim(fmt.Sprintf("… and %d more (use -n or --all)", len(tasks)-limit)))
		}

		return nil
	},
}

func init() {
	listCmd.Flags().StringP("status", "s", "", "filter by status (open, in_progress, done, cancelled)")
	listCmd.Flags().StringP("type", "t", "", "filter by type (feature, bug, task, epic)")
	listCmd.Flags().String("tag", "", "filter by tag")
	listCmd.Flags().Bool("all", false, "include done and cancelled tasks")
	listCmd.Flags().IntP("number", "n", 0, "max number of tasks to show")
	AddRootFlag(listCmd)
	AddJSONFlag(listCmd)
	rootCmd.AddCommand(listCmd)
}
