package cmd

import (
	"fmt"
	"strings"

	"github.com/spf13/cobra"

	"vine/store"
	"vine/utils"
)

var statusCmd = &cobra.Command{
	Use:   "status",
	Short: "Show a general summary of all tasks",
	RunE: func(cmd *cobra.Command, args []string) error {
		s := GetStore(cmd)
		detailed, _ := cmd.Flags().GetBool("detailed")

		projectName := getProjectName(cmd)

		if detailed {
			return statusDetailed(cmd, s, projectName)
		}
		return statusSimple(cmd, s, projectName)
	},
}

func statusSimple(cmd *cobra.Command, s *store.Store, project string) error {
	counts, err := s.TaskSummary()
	if err != nil {
		return err
	}

	total := 0
	for _, c := range counts {
		total += c.Count
	}

	if IsJSON(cmd) {
		PrintOutput(cmd, "", map[string]any{
			"project": project,
			"total":   total,
			"status":  counts,
		})
		return nil
	}

	if total == 0 {
		fmt.Printf("%s: no tasks yet\n", utils.Bold(project))
		return nil
	}

	fmt.Printf("%s: %d tasks\n\n", utils.Bold(project), total)
	for _, c := range counts {
		label := utils.StatusColor(c.Status).Sprintf("%-14s", c.Status)
		fmt.Printf("  %s %d\n", label, c.Count)
	}

	return nil
}

func statusDetailed(cmd *cobra.Command, s *store.Store, project string) error {
	detailed, err := s.TaskSummaryDetailed()
	if err != nil {
		return err
	}

	// Also get totals per status.
	counts, err := s.TaskSummary()
	if err != nil {
		return err
	}

	total := 0
	for _, c := range counts {
		total += c.Count
	}

	if IsJSON(cmd) {
		PrintOutput(cmd, "", map[string]any{
			"project":  project,
			"total":    total,
			"status":   counts,
			"detailed": detailed,
		})
		return nil
	}

	if total == 0 {
		fmt.Printf("%s: no tasks yet\n", utils.Bold(project))
		return nil
	}

	// Group detailed counts by status.
	byStatus := make(map[string][]store.StatusTypeCount)
	for _, d := range detailed {
		byStatus[d.Status] = append(byStatus[d.Status], d)
	}

	fmt.Printf("%s: %d tasks\n\n", utils.Bold(project), total)
	for _, c := range counts {
		label := utils.StatusColor(c.Status).Sprintf("%-14s", c.Status)
		breakdown := formatTypeBreakdown(byStatus[c.Status])
		fmt.Printf("  %s %d  %s\n", label, c.Count, utils.Dim(breakdown))
	}

	return nil
}

func formatTypeBreakdown(types []store.StatusTypeCount) string {
	if len(types) == 0 {
		return ""
	}
	parts := make([]string, len(types))
	for i, t := range types {
		parts[i] = fmt.Sprintf("%d %s", t.Count, t.Type)
	}
	return "(" + strings.Join(parts, ", ") + ")"
}

func init() {
	statusCmd.Flags().Bool("detailed", false, "show breakdown by type")
	AddJSONFlag(statusCmd)
	rootCmd.AddCommand(statusCmd)
}
