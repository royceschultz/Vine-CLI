package cmd

import (
	"fmt"

	"github.com/spf13/cobra"

	"vine/utils"
)

var dependencyCmd = &cobra.Command{
	Use:     "dependency",
	Aliases: []string{"dep"},
	Short:   "Manage dependencies between tasks",
}

var dependencyAddCmd = &cobra.Command{
	Use:   "add <task-id> <depends-on-id>",
	Short: "Mark a task as depending on another task",
	Long:  "The first task will be blocked until the second task is done or cancelled.",
	Args:  cobra.ExactArgs(2),
	RunE: func(cmd *cobra.Command, args []string) error {
		s := GetStore(cmd)

		_, taskID := utils.ParseTaskID(args[0])
		_, depID := utils.ParseTaskID(args[1])

		// Verify both tasks exist.
		task, err := s.GetTask(taskID)
		if err != nil {
			return fmt.Errorf("task %q not found", args[0])
		}
		dep, err := s.GetTask(depID)
		if err != nil {
			return fmt.Errorf("task %q not found", args[1])
		}

		if taskID == depID {
			return fmt.Errorf("a task cannot depend on itself")
		}

		if err := s.AddDependency(taskID, depID); err != nil {
			return err
		}

		projectName := getProjectName(cmd)
		displayTask := utils.FormatTaskID(projectName, task.ID)
		displayDep := utils.FormatTaskID(projectName, dep.ID)

		PrintOutput(cmd, fmt.Sprintf("%s now depends on %s", displayTask, displayDep), map[string]string{
			"task_id":       task.ID,
			"depends_on_id": dep.ID,
		})
		return nil
	},
}

var dependencyRemoveCmd = &cobra.Command{
	Use:   "remove <task-id> <depends-on-id>",
	Short: "Remove a dependency between tasks",
	Args:  cobra.ExactArgs(2),
	RunE: func(cmd *cobra.Command, args []string) error {
		s := GetStore(cmd)

		_, taskID := utils.ParseTaskID(args[0])
		_, depID := utils.ParseTaskID(args[1])

		if err := s.RemoveDependency(taskID, depID); err != nil {
			return err
		}

		projectName := getProjectName(cmd)
		displayTask := utils.FormatTaskID(projectName, taskID)
		displayDep := utils.FormatTaskID(projectName, depID)

		PrintOutput(cmd, fmt.Sprintf("%s no longer depends on %s", displayTask, displayDep), map[string]string{
			"task_id":       taskID,
			"depends_on_id": depID,
		})
		return nil
	},
}

var dependencyListCmd = &cobra.Command{
	Use:   "list <task-id>",
	Short: "List dependencies of a task",
	Args:  cobra.ExactArgs(1),
	RunE: func(cmd *cobra.Command, args []string) error {
		s := GetStore(cmd)

		_, taskID := utils.ParseTaskID(args[0])

		deps, err := s.DependenciesOf(taskID)
		if err != nil {
			return err
		}

		if IsJSON(cmd) {
			tasks := make([]any, 0, len(deps))
			for _, d := range deps {
				if t, err := s.GetTask(d.DependsOnID); err == nil {
					tasks = append(tasks, t)
				}
			}
			PrintOutput(cmd, "", tasks)
			return nil
		}

		if len(deps) == 0 {
			fmt.Println("No dependencies.")
			return nil
		}

		projectName := getProjectName(cmd)

		for _, d := range deps {
			depTask, err := s.GetTask(d.DependsOnID)
			if err != nil {
				continue
			}
			displayID := utils.FormatTaskID(projectName, depTask.ID)
			statusLabel := utils.StatusColor(depTask.Status).Sprint(depTask.Status)
			fmt.Printf("  %s  %s  %s\n", utils.Dim(displayID), statusLabel, depTask.Name)
		}

		return nil
	},
}

var dependencyDependentsCmd = &cobra.Command{
	Use:   "dependents <task-id>",
	Short: "List tasks that are waiting on this task",
	Args:  cobra.ExactArgs(1),
	RunE: func(cmd *cobra.Command, args []string) error {
		s := GetStore(cmd)

		_, taskID := utils.ParseTaskID(args[0])

		deps, err := s.DependentsOf(taskID)
		if err != nil {
			return err
		}

		if IsJSON(cmd) {
			tasks := make([]any, 0, len(deps))
			for _, d := range deps {
				if t, err := s.GetTask(d.TaskID); err == nil {
					tasks = append(tasks, t)
				}
			}
			PrintOutput(cmd, "", tasks)
			return nil
		}

		if len(deps) == 0 {
			fmt.Println("No dependents.")
			return nil
		}

		projectName := getProjectName(cmd)

		for _, d := range deps {
			depTask, err := s.GetTask(d.TaskID)
			if err != nil {
				continue
			}
			displayID := utils.FormatTaskID(projectName, depTask.ID)
			statusLabel := utils.StatusColor(depTask.Status).Sprint(depTask.Status)
			fmt.Printf("  %s  %s  %s\n", utils.Dim(displayID), statusLabel, depTask.Name)
		}

		return nil
	},
}

func init() {
	AddJSONFlag(dependencyAddCmd)
	AddJSONFlag(dependencyRemoveCmd)
	AddJSONFlag(dependencyListCmd)
	AddJSONFlag(dependencyDependentsCmd)
	dependencyCmd.AddCommand(dependencyAddCmd)
	dependencyCmd.AddCommand(dependencyRemoveCmd)
	dependencyCmd.AddCommand(dependencyListCmd)
	dependencyCmd.AddCommand(dependencyDependentsCmd)
	rootCmd.AddCommand(dependencyCmd)
}
