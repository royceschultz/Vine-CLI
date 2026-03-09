package cmd

import (
	"fmt"

	"github.com/spf13/cobra"

	"vine/store"
	"vine/utils"
)

var readyCmd = &cobra.Command{
	Use:   "ready",
	Short: "List tasks that are ready to work on",
	RunE: func(cmd *cobra.Command, args []string) error {
		var tasks []store.Task
		var err error

		if IsRemote(cmd) {
			c, project := GetRemoteClient(cmd)
			tasks, err = c.ReadyTasks(project)
		} else {
			s := GetStore(cmd)
			tasks, err = s.ReadyTasks()
		}
		if err != nil {
			return err
		}

		// Filter to root-only if requested.
		if IsRoot(cmd) {
			tasks = filterRoot(tasks)
		}

		// All tasks from ReadyTasks() are ready by definition.
		for i := range tasks {
			tasks[i].Status = "ready"
		}

		n, _ := cmd.Flags().GetInt("number")
		showAll, _ := cmd.Flags().GetBool("all")

		if IsJSON(cmd) {
			PrintOutput(cmd, "", tasks)
			return nil
		}

		if len(tasks) == 0 {
			fmt.Println("No tasks ready.")
			return nil
		}

		projectName := getProjectName(cmd)

		var counts map[string]int
		var parents map[string]*store.Task
		if !IsRemote(cmd) {
			s := GetStore(cmd)
			counts = collectChildCounts(s, tasks)
			parents = collectParents(s, tasks)
		}

		limit := len(tasks)
		if !showAll && n > 0 && n < limit {
			limit = n
		}

		fmt.Printf("%s ready:\n\n", utils.Bold(fmt.Sprintf("%d", len(tasks))))
		for i := 0; i < limit; i++ {
			t := tasks[i]
			displayID := utils.FormatTaskID(projectName, t.ID)
			typeLabel := ""
			if t.Type != "task" {
				typeLabel = " " + utils.Dim("["+t.Type+"]")
			}

			pLabel := parentLabel(projectName, parents, t.ParentID)
			subLabel := subtaskLabel(counts, t.ID)

			fmt.Printf("  %s  %s%s%s%s\n", utils.Dim(displayID), t.Name, typeLabel, pLabel, subLabel)

			if t.Description != "" {
				fmt.Printf("         %s\n", utils.Dim(t.Description))
			}
		}

		if limit < len(tasks) {
			fmt.Printf("\n  %s\n", utils.Dim(fmt.Sprintf("… and %d more (use --all to show all)", len(tasks)-limit)))
		}

		return nil
	},
}

// filterRoot returns only tasks with no parent.
func filterRoot(tasks []store.Task) []store.Task {
	var filtered []store.Task
	for _, t := range tasks {
		if t.ParentID == nil {
			filtered = append(filtered, t)
		}
	}
	return filtered
}

func init() {
	readyCmd.Flags().IntP("number", "n", 10, "max number of tasks to show")
	readyCmd.Flags().Bool("all", false, "show all ready tasks")
	AddRootFlag(readyCmd)
	AddJSONFlag(readyCmd)
	rootCmd.AddCommand(readyCmd)
}
