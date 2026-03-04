package cmd

import (
	"fmt"
	"os"
	"os/exec"
	"strings"

	"github.com/spf13/cobra"

	"vine/config"
	"vine/store"
	"vine/utils"
)

var createCmd = &cobra.Command{
	Use:   "create <name>",
	Short: "Create a new task",
	Args:  cobra.MinimumNArgs(1),
	RunE: func(cmd *cobra.Command, args []string) error {
		s := GetStore(cmd)

		name := strings.Join(args, " ")
		description, _ := cmd.Flags().GetString("description")
		details, _ := cmd.Flags().GetString("details")
		taskType, _ := cmd.Flags().GetString("type")
		parentID, _ := cmd.Flags().GetString("parent")
		tagsFlag, _ := cmd.Flags().GetStringSlice("tag")

		var parentPtr *string
		if cmd.Flags().Changed("parent") {
			_, bareID := utils.ParseTaskID(parentID)
			parentPtr = &bareID
		}

		meta := detectMetadata()

		task, err := s.CreateTask(store.CreateTaskParams{
			Name:        name,
			Description: description,
			Details:     details,
			Type:        taskType,
			ParentID:    parentPtr,
			Metadata:    meta,
		})
		if err != nil {
			return err
		}

		for _, tag := range tagsFlag {
			if err := s.AddTag(task.ID, tag); err != nil {
				return err
			}
		}

		if description == "" && !IsJSON(cmd) {
			fmt.Fprintln(os.Stderr, "tip: add -d \"description\" to help others understand this task")
		}

		projectName := getProjectName(cmd)
		displayID := utils.FormatTaskID(projectName, task.ID)
		PrintOutput(cmd, fmt.Sprintf("created task %s: %s", displayID, task.Name), task)

		return nil
	},
}

// detectMetadata gathers environment context (branch, directory).
func detectMetadata() *store.TaskMetadata {
	meta := &store.TaskMetadata{}

	if dir, err := os.Getwd(); err == nil {
		meta.CreatedDir = dir
		meta.UpdatedDir = dir
	}

	if branch, err := exec.Command("git", "rev-parse", "--abbrev-ref", "HEAD").Output(); err == nil {
		b := strings.TrimSpace(string(branch))
		meta.CreatedBranch = b
		meta.UpdatedBranch = b
	}

	return meta
}

// getProjectName resolves the project name from config.
func getProjectName(cmd *cobra.Command) string {
	cwd, err := os.Getwd()
	if err != nil {
		return "vine"
	}
	projectRoot, err := config.FindProjectRoot(cwd)
	if err != nil {
		return "vine"
	}
	cfg, err := config.Load(projectRoot)
	if err != nil {
		return "vine"
	}
	return config.ProjectName(projectRoot, cfg)
}

func init() {
	createCmd.Flags().StringP("description", "d", "", "brief task description")
	createCmd.Flags().String("details", "", "detailed technical information")
	createCmd.Flags().StringP("type", "t", "task", "task type: feature, bug, task, or epic")
	createCmd.Flags().StringP("parent", "p", "", "parent task ID")
	createCmd.Flags().StringSlice("tag", nil, "tags (comma-separated or repeated)")
	AddJSONFlag(createCmd)
	rootCmd.AddCommand(createCmd)
}
