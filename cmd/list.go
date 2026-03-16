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
		status, _ := cmd.Flags().GetString("status")
		taskType, _ := cmd.Flags().GetString("type")
		tag, _ := cmd.Flags().GetString("tag")
		grep, _ := cmd.Flags().GetString("grep")
		showAll, _ := cmd.Flags().GetBool("all")
		n, _ := cmd.Flags().GetInt("number")

		filter := store.TaskFilter{
			Status:   status,
			Type:     taskType,
			Tag:      tag,
			Grep:     grep,
			All:      showAll,
			RootOnly: IsRoot(cmd),
		}

		var enriched []store.TaskWithDeps
		var plainTasks []store.Task
		var err error

		if IsRemote(cmd) {
			c, project := GetRemoteClient(cmd)
			enriched, err = c.ListTasks(project, filter)
		} else {
			s := GetStore(cmd)
			plainTasks, err = s.ListTasksFiltered(filter)
		}
		if err != nil {
			return err
		}

		// For local mode, enrich tasks with dependency IDs and effective status.
		if !IsRemote(cmd) {
			s := GetStore(cmd)
			s.EnrichEffectiveStatus(plainTasks)

			// Post-filter for effective statuses (ready/blocked) which can't be
			// filtered at the SQL level since they're computed from dependencies.
			if filter.IsEffectiveStatus() {
				filtered := plainTasks[:0]
				for _, t := range plainTasks {
					if t.Status == filter.Status {
						filtered = append(filtered, t)
					}
				}
				plainTasks = filtered
			}

			ids := make([]string, len(plainTasks))
			for i, t := range plainTasks {
				ids[i] = t.ID
			}
			dependsOn, blocks, _ := s.DependencyIDsForTasks(ids)
			enriched = make([]store.TaskWithDeps, len(plainTasks))
			for i, t := range plainTasks {
				enriched[i] = store.TaskWithDeps{
					Task:         t,
					DependsOnIDs: emptyIfNil(dependsOn[t.ID]),
					BlocksIDs:    emptyIfNil(blocks[t.ID]),
				}
			}
		}

		if IsJSON(cmd) {
			PrintOutput(cmd, "", enriched)
			return nil
		}

		if len(enriched) == 0 {
			fmt.Println("No tasks found.")
			return nil
		}

		projectName := getProjectName(cmd)

		// Extract plain tasks for display helpers.
		tasks := make([]store.Task, len(enriched))
		for i, e := range enriched {
			tasks[i] = e.Task
		}

		var counts map[string]int
		var parents map[string]*store.Task
		if !IsRemote(cmd) {
			s := GetStore(cmd)
			counts = collectChildCounts(s, tasks)
			parents = collectParents(s, tasks)
		}

		limit := len(enriched)
		if n > 0 && n < limit {
			limit = n
		}

		for i := 0; i < limit; i++ {
			t := enriched[i].Task
			displayID := utils.FormatTaskID(projectName, t.ID)

			statusLabel := utils.StatusColor(t.Status).Sprintf("%-12s", t.Status)

			pLabel := parentLabel(projectName, parents, t.ParentID)
			subLabel := subtaskLabel(counts, t.ID)

			fmt.Printf("  %s  %s  %s%s%s%s\n", utils.Dim(displayID), statusLabel, t.Name, utils.TypeLabel(t.Type), pLabel, subLabel)
		}

		if limit < len(enriched) {
			fmt.Printf("\n  %s\n", utils.Dim(fmt.Sprintf("… and %d more (use -n or --all)", len(enriched)-limit)))
		}

		return nil
	},
}

func init() {
	listCmd.Flags().StringP("status", "s", "", "filter by status (ready, blocked, open, in_progress, done, cancelled)")
	listCmd.Flags().StringP("type", "t", "", "filter by type (feature, bug, task, epic)")
	listCmd.Flags().String("tag", "", "filter by tag")
	listCmd.Flags().String("grep", "", "filter by substring match on task name")
	listCmd.Flags().Bool("all", false, "include done and cancelled tasks")
	listCmd.Flags().IntP("number", "n", 0, "max number of tasks to show")
	AddRootFlag(listCmd)
	AddJSONFlag(listCmd)
	rootCmd.AddCommand(listCmd)
}
