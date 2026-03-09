package cmd

import (
	"fmt"
	"strings"

	"github.com/spf13/cobra"

	"vine/store"
	"vine/utils"
)

var searchCmd = &cobra.Command{
	Use:   "search <query>",
	Short: "Search tasks by keyword",
	Long:  "Searches task names, descriptions, and details for the given keyword.",
	Args:  cobra.MinimumNArgs(1),
	RunE: func(cmd *cobra.Command, args []string) error {
		query := strings.Join(args, " ")

		var tasks []store.Task
		var err error

		if IsRemote(cmd) {
			c, project := GetRemoteClient(cmd)
			tasks, err = c.SearchTasks(project, query)
		} else {
			s := GetStore(cmd)
			tasks, err = s.SearchTasks(query)
		}
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
			fmt.Printf("No tasks matching %q.\n", query)
			return nil
		}

		projectName := getProjectName(cmd)

		var counts map[string]int
		if !IsRemote(cmd) {
			s := GetStore(cmd)
			counts = collectChildCounts(s, tasks)
		}

		fmt.Printf("%d result(s) for %q:\n\n", len(tasks), query)
		for _, t := range tasks {
			displayID := utils.FormatTaskID(projectName, t.ID)
			statusLabel := utils.StatusColor(t.Status).Sprint(t.Status)
			subLabel := subtaskLabel(counts, t.ID)
			fmt.Printf("  %s  %s  %s%s%s\n", utils.Dim(displayID), statusLabel, t.Name, utils.TypeLabel(t.Type), subLabel)
		}

		return nil
	},
}

func init() {
	AddRootFlag(searchCmd)
	AddJSONFlag(searchCmd)
	rootCmd.AddCommand(searchCmd)
}
