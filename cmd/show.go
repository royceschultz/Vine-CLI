package cmd

import (
	"fmt"
	"strings"

	"github.com/spf13/cobra"

	"vine/utils"
)

var showCmd = &cobra.Command{
	Use:   "show <task-id>",
	Short: "Show detailed information about a task",
	Args:  cobra.ExactArgs(1),
	RunE: func(cmd *cobra.Command, args []string) error {
		s := GetStore(cmd)
		projectName := getProjectName(cmd)
		detailed, _ := cmd.Flags().GetBool("detailed")
		short, _ := cmd.Flags().GetBool("short")

		_, bareID := utils.ParseTaskID(args[0])

		task, err := s.GetTask(bareID)
		if err != nil {
			return fmt.Errorf("task %q not found", args[0])
		}

		// JSON: --short returns just the task row, default adds relations.
		if IsJSON(cmd) {
			if short {
				PrintOutput(cmd, "", task)
				return nil
			}

			data := map[string]any{
				"task": task,
			}

			if task.ParentID != nil {
				ancestors, _ := s.AncestorChain(bareID)
				if len(ancestors) > 0 {
					data["ancestors"] = ancestors
				}
			}

			children, _ := s.ChildTasks(bareID)
			if len(children) > 0 {
				data["subtasks"] = children
			}

			deps, _ := s.DependenciesOf(bareID)
			if len(deps) > 0 {
				var depTasks []any
				for _, d := range deps {
					if t, err := s.GetTask(d.DependsOnID); err == nil {
						depTasks = append(depTasks, t)
					}
				}
				data["depends_on"] = depTasks
			}

			dependents, _ := s.DependentsOf(bareID)
			if len(dependents) > 0 {
				var depTasks []any
				for _, d := range dependents {
					if t, err := s.GetTask(d.TaskID); err == nil {
						depTasks = append(depTasks, t)
					}
				}
				data["blocks"] = depTasks
			}

			tags, _ := s.TagsForTask(bareID)
			if len(tags) > 0 {
				data["tags"] = tags
			}

			if detailed {
				comments, _ := s.CommentsForTask(bareID)
				if len(comments) > 0 {
					data["comments"] = comments
				}
			}

			PrintOutput(cmd, "", data)
			return nil
		}

		// Short: just the essentials.
		if short {
			displayID := utils.FormatTaskID(projectName, task.ID)
			statusLabel := utils.StatusColor(task.Status).Sprint(task.Status)
			typeLabel := ""
			if task.Type != "task" {
				typeLabel = " [" + task.Type + "]"
			}
			fmt.Printf("%s  %s  %s%s\n", displayID, statusLabel, task.Name, typeLabel)
			if task.Description != "" {
				fmt.Printf("  %s\n", task.Description)
			}
			return nil
		}

		// Header.
		displayID := utils.FormatTaskID(projectName, task.ID)
		statusLabel := utils.StatusColor(task.Status).Sprint(task.Status)
		fmt.Printf("%s  %s  %s\n", utils.Bold(displayID), statusLabel, utils.Bold(task.Name))

		// Type (if not default).
		if task.Type != "task" {
			fmt.Printf("  type:  %s\n", task.Type)
		}

		// Description.
		if task.Description != "" {
			fmt.Printf("\n  %s\n", task.Description)
		}

		// Details.
		if task.Details != "" {
			fmt.Printf("\n  %s\n  %s\n", utils.Dim("details:"), task.Details)
		}

		// Close reason (for done/cancelled tasks).
		if task.Status == "done" || task.Status == "cancelled" {
			if reason, err := s.LatestCloseReason(bareID); err == nil {
				fmt.Printf("\n  %s %s\n", utils.Dim("reason:"), reason.Content)
			}
		}

		// Tags.
		tags, _ := s.TagsForTask(bareID)
		if len(tags) > 0 {
			names := make([]string, len(tags))
			for i, t := range tags {
				names[i] = t.Name
			}
			fmt.Printf("\n  %s  %s\n", utils.Dim("tags:"), strings.Join(names, ", "))
		}

		// Ancestor chain (immediate parent → root).
		if task.ParentID != nil {
			ancestors, _ := s.AncestorChain(bareID)
			if len(ancestors) > 0 {
				fmt.Printf("\n  %s  ", utils.Dim("parent:"))
				for i, a := range ancestors {
					aDisplay := utils.FormatTaskID(projectName, a.ID)
					if i > 0 {
						fmt.Printf(" > ")
					}
					fmt.Printf("%s %s", utils.Dim(aDisplay), a.Name)
				}
				fmt.Println()
			}
		}

		// Subtasks.
		children, _ := s.ChildTasks(bareID)
		if len(children) > 0 {
			fmt.Printf("\n  %s\n", utils.Dim("subtasks:"))
			for _, c := range children {
				childDisplay := utils.FormatTaskID(projectName, c.ID)
				status := utils.StatusColor(c.Status).Sprint(c.Status)
				fmt.Printf("    %s  %s  %s\n", utils.Dim(childDisplay), status, c.Name)
			}
		}

		// Dependencies (what this task depends on).
		deps, _ := s.DependenciesOf(bareID)
		if len(deps) > 0 {
			fmt.Printf("\n  %s\n", utils.Dim("depends on:"))
			for _, d := range deps {
				if t, err := s.GetTask(d.DependsOnID); err == nil {
					depDisplay := utils.FormatTaskID(projectName, t.ID)
					status := utils.StatusColor(t.Status).Sprint(t.Status)
					fmt.Printf("    %s  %s  %s\n", utils.Dim(depDisplay), status, t.Name)
				}
			}
		}

		// Dependents (what tasks this one blocks).
		dependents, _ := s.DependentsOf(bareID)
		if len(dependents) > 0 {
			fmt.Printf("\n  %s\n", utils.Dim("blocks:"))
			for _, d := range dependents {
				if t, err := s.GetTask(d.TaskID); err == nil {
					depDisplay := utils.FormatTaskID(projectName, t.ID)
					status := utils.StatusColor(t.Status).Sprint(t.Status)
					fmt.Printf("    %s  %s  %s\n", utils.Dim(depDisplay), status, t.Name)
				}
			}
		}

		// Timestamps.
		fmt.Printf("\n  %s %s    %s %s\n", utils.Dim("created:"), task.CreatedAt, utils.Dim("updated:"), task.UpdatedAt)

		// Detailed: metadata + comments.
		if detailed {
			meta, _ := task.ParseMetadata()
			if meta != nil {
				var parts []string
				if meta.CreatedBranch != "" {
					parts = append(parts, "branch: "+meta.CreatedBranch)
				}
				if meta.CreatedDir != "" {
					parts = append(parts, "dir: "+meta.CreatedDir)
				}
				if len(parts) > 0 {
					fmt.Printf("\n  %s  %s\n", utils.Dim("created from:"), strings.Join(parts, "  "))
				}
				parts = nil
				if meta.UpdatedBranch != "" {
					parts = append(parts, "branch: "+meta.UpdatedBranch)
				}
				if meta.UpdatedDir != "" {
					parts = append(parts, "dir: "+meta.UpdatedDir)
				}
				if len(parts) > 0 {
					fmt.Printf("  %s  %s\n", utils.Dim("updated from:"), strings.Join(parts, "  "))
				}
			}

			comments, _ := s.CommentsForTask(bareID)
			if len(comments) > 0 {
				fmt.Printf("\n  %s\n", utils.Dim("comments:"))
				for _, c := range comments {
					typeLabel := ""
					if c.Type != "comment" {
						typeLabel = utils.Dim(fmt.Sprintf(" [%s]", c.Type))
					}
					fmt.Printf("    %s%s  %s\n", utils.Dim(c.CreatedAt), typeLabel, c.Content)
				}
			}
		}

		return nil
	},
}

func init() {
	showCmd.Flags().Bool("detailed", false, "include metadata and comments")
	showCmd.Flags().Bool("short", false, "minimal output (id, status, name, description)")
	AddJSONFlag(showCmd)
	rootCmd.AddCommand(showCmd)
}
