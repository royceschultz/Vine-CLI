package cmd

import (
	"fmt"
	"strings"

	"github.com/spf13/cobra"

	"vine/store"
	"vine/utils"
)

// showData holds all the data needed to render a task's show view.
// This abstracts over local store vs remote client.
type showData struct {
	task       *store.Task
	ancestors  []store.Task
	children   []store.Task
	deps       []store.Dependency
	depTasks   []*store.Task
	dependents []store.Dependency
	blockTasks []*store.Task
	tags       []store.Tag
	comments   []store.Comment
	closeReason *store.Comment
}

func fetchShowDataLocal(s *store.Store, bareID string, detailed bool) *showData {
	d := &showData{}
	d.task, _ = s.GetTask(bareID)
	if d.task == nil {
		return nil
	}
	if d.task.ParentID != nil {
		d.ancestors, _ = s.AncestorChain(bareID)
	}
	d.children, _ = s.ChildTasks(bareID)
	d.deps, _ = s.DependenciesOf(bareID)
	for _, dep := range d.deps {
		if t, err := s.GetTask(dep.DependsOnID); err == nil {
			d.depTasks = append(d.depTasks, t)
		}
	}
	d.dependents, _ = s.DependentsOf(bareID)
	for _, dep := range d.dependents {
		if t, err := s.GetTask(dep.TaskID); err == nil {
			d.blockTasks = append(d.blockTasks, t)
		}
	}
	d.tags, _ = s.TagsForTask(bareID)

	// Enrich with effective status (open → blocked/ready).
	mainTask := []store.Task{*d.task}
	s.EnrichEffectiveStatus(mainTask)
	d.task.Status = mainTask[0].Status

	s.EnrichEffectiveStatus(d.children)
	s.EnrichEffectiveStatusPtr(d.depTasks)
	s.EnrichEffectiveStatusPtr(d.blockTasks)

	if d.task.Status == "done" || d.task.Status == "cancelled" {
		d.closeReason, _ = s.LatestCloseReason(bareID)
	}
	if detailed {
		d.comments, _ = s.CommentsForTask(bareID)
	}
	return d
}

func fetchShowDataRemote(cmd *cobra.Command, bareID string, detailed bool) *showData {
	c, project := GetRemoteClient(cmd)
	d := &showData{}
	var err error
	d.task, err = c.GetTask(project, bareID)
	if err != nil {
		return nil
	}
	if d.task.ParentID != nil {
		d.ancestors, _ = c.AncestorChain(project, bareID)
	}
	d.children, _ = c.ChildTasks(project, bareID)
	d.deps, _ = c.DependenciesOf(project, bareID)
	for _, dep := range d.deps {
		if t, err := c.GetTask(project, dep.DependsOnID); err == nil {
			d.depTasks = append(d.depTasks, t)
		}
	}
	d.dependents, _ = c.DependentsOf(project, bareID)
	for _, dep := range d.dependents {
		if t, err := c.GetTask(project, dep.TaskID); err == nil {
			d.blockTasks = append(d.blockTasks, t)
		}
	}
	d.tags, _ = c.TagsForTask(project, bareID)
	if detailed {
		d.comments, _ = c.CommentsForTask(project, bareID)
	}
	return d
}

func renderShow(cmd *cobra.Command, projectName string, d *showData, detailed, short, verbose bool) {
	task := d.task

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
		return
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
	if d.closeReason != nil {
		fmt.Printf("\n  %s %s\n", utils.Dim("reason:"), d.closeReason.Content)
	}

	// Tags.
	if len(d.tags) > 0 {
		names := make([]string, len(d.tags))
		for i, t := range d.tags {
			names[i] = t.Name
		}
		fmt.Printf("\n  %s  %s\n", utils.Dim("tags:"), strings.Join(names, ", "))
	}

	// Ancestor chain (immediate parent → root).
	if len(d.ancestors) > 0 {
		fmt.Printf("\n  %s  ", utils.Dim("parent:"))
		for i, a := range d.ancestors {
			aDisplay := utils.FormatTaskID(projectName, a.ID)
			if i > 0 {
				fmt.Printf(" > ")
			}
			fmt.Printf("%s %s", utils.Dim(aDisplay), a.Name)
		}
		fmt.Println()
	}

	// Subtasks.
	if len(d.children) > 0 {
		fmt.Printf("\n  %s\n", utils.Dim("subtasks:"))
		for _, c := range d.children {
			childDisplay := utils.FormatTaskID(projectName, c.ID)
			status := utils.StatusColor(c.Status).Sprint(c.Status)
			fmt.Printf("    %s  %s  %s\n", utils.Dim(childDisplay), status, c.Name)
		}
	}

	// Dependencies (what this task depends on).
	if len(d.depTasks) > 0 {
		fmt.Printf("\n  %s\n", utils.Dim("depends on:"))
		for _, t := range d.depTasks {
			depDisplay := utils.FormatTaskID(projectName, t.ID)
			status := utils.StatusColor(t.Status).Sprint(t.Status)
			fmt.Printf("    %s  %s  %s\n", utils.Dim(depDisplay), status, t.Name)
		}
	}

	// Dependents (what tasks this one blocks).
	if len(d.blockTasks) > 0 {
		fmt.Printf("\n  %s\n", utils.Dim("blocks:"))
		for _, t := range d.blockTasks {
			depDisplay := utils.FormatTaskID(projectName, t.ID)
			status := utils.StatusColor(t.Status).Sprint(t.Status)
			fmt.Printf("    %s  %s  %s\n", utils.Dim(depDisplay), status, t.Name)
		}
	}

	// Timestamps (only with --verbose).
	if verbose {
		fmt.Printf("\n  %s %s    %s %s\n", utils.Dim("created:"), task.CreatedAt, utils.Dim("updated:"), task.UpdatedAt)
	}

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

		if len(d.comments) > 0 {
			fmt.Printf("\n  %s\n", utils.Dim("comments:"))
			for _, c := range d.comments {
				typeLabel := ""
				if c.Type != "comment" {
					typeLabel = utils.Dim(fmt.Sprintf(" [%s]", c.Type))
				}
				fmt.Printf("    %s%s  %s\n", utils.Dim(c.CreatedAt), typeLabel, c.Content)
			}
		}
	}
}

var showCmd = &cobra.Command{
	Use:   "show <task-id> [task-id...]",
	Short: "Show detailed information about one or more tasks",
	Args:  cobra.MinimumNArgs(1),
	RunE: func(cmd *cobra.Command, args []string) error {
		projectName := getProjectName(cmd)
		detailed, _ := cmd.Flags().GetBool("detailed")
		short, _ := cmd.Flags().GetBool("short")
		verbose, _ := cmd.Flags().GetBool("verbose")

		var allData []*showData
		for _, arg := range args {
			_, bareID := utils.ParseTaskID(arg)

			var d *showData
			if IsRemote(cmd) {
				d = fetchShowDataRemote(cmd, bareID, detailed)
			} else {
				s := GetStore(cmd)
				d = fetchShowDataLocal(s, bareID, detailed)
			}
			if d == nil {
				return fmt.Errorf("task %q not found", arg)
			}
			allData = append(allData, d)
		}

		if IsJSON(cmd) {
			if len(allData) == 1 {
				d := allData[0]
				if short {
					PrintOutput(cmd, "", d.task)
					return nil
				}
				data := map[string]any{"task": d.task}
				if len(d.ancestors) > 0 {
					data["ancestors"] = d.ancestors
				}
				if len(d.children) > 0 {
					data["subtasks"] = d.children
				}
				if len(d.depTasks) > 0 {
					data["depends_on"] = d.depTasks
				}
				if len(d.blockTasks) > 0 {
					data["blocks"] = d.blockTasks
				}
				if len(d.tags) > 0 {
					data["tags"] = d.tags
				}
				if len(d.comments) > 0 {
					data["comments"] = d.comments
				}
				PrintOutput(cmd, "", data)
			} else {
				var results []any
				for _, d := range allData {
					if short {
						results = append(results, d.task)
						continue
					}
					data := map[string]any{"task": d.task}
					if len(d.ancestors) > 0 {
						data["ancestors"] = d.ancestors
					}
					if len(d.children) > 0 {
						data["subtasks"] = d.children
					}
					if len(d.depTasks) > 0 {
						data["depends_on"] = d.depTasks
					}
					if len(d.blockTasks) > 0 {
						data["blocks"] = d.blockTasks
					}
					if len(d.tags) > 0 {
						data["tags"] = d.tags
					}
					if len(d.comments) > 0 {
						data["comments"] = d.comments
					}
					results = append(results, data)
				}
				PrintOutput(cmd, "", results)
			}
			return nil
		}

		for i, d := range allData {
			if i > 0 {
				fmt.Println()
				fmt.Println(strings.Repeat("─", 40))
				fmt.Println()
			}
			renderShow(cmd, projectName, d, detailed, short, verbose)
		}

		return nil
	},
}

func init() {
	showCmd.Flags().Bool("detailed", false, "include metadata and comments")
	showCmd.Flags().Bool("short", false, "minimal output (id, status, name, description)")
	showCmd.Flags().BoolP("verbose", "v", false, "include timestamps")
	AddJSONFlag(showCmd)
	rootCmd.AddCommand(showCmd)
}
