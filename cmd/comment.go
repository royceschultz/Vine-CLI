package cmd

import (
	"fmt"
	"strconv"
	"strings"

	"github.com/spf13/cobra"

	"vine/utils"
)

var commentCmd = &cobra.Command{
	Use:   "comment",
	Short: "Manage task comments",
}

var commentAddCmd = &cobra.Command{
	Use:   "add <task-id> <message>",
	Short: "Add a comment to a task",
	Args:  cobra.MinimumNArgs(2),
	RunE: func(cmd *cobra.Command, args []string) error {
		s := GetStore(cmd)

		_, bareID := utils.ParseTaskID(args[0])

		if _, err := s.GetTask(bareID); err != nil {
			return fmt.Errorf("task %q not found", args[0])
		}

		content := strings.Join(args[1:], " ")

		comment, err := s.AddComment(bareID, "comment", content)
		if err != nil {
			return err
		}

		projectName := getProjectName(cmd)
		displayID := utils.FormatTaskID(projectName, bareID)
		PrintOutput(cmd, fmt.Sprintf("added comment to %s", displayID), comment)

		return nil
	},
}

var commentListCmd = &cobra.Command{
	Use:   "list <task-id>",
	Short: "List comments on a task",
	Args:  cobra.ExactArgs(1),
	RunE: func(cmd *cobra.Command, args []string) error {
		s := GetStore(cmd)

		_, bareID := utils.ParseTaskID(args[0])

		if _, err := s.GetTask(bareID); err != nil {
			return fmt.Errorf("task %q not found", args[0])
		}

		comments, err := s.CommentsForTask(bareID)
		if err != nil {
			return err
		}

		if IsJSON(cmd) {
			PrintOutput(cmd, "", comments)
			return nil
		}

		if len(comments) == 0 {
			fmt.Println("No comments.")
			return nil
		}

		for _, c := range comments {
			typeLabel := ""
			if c.Type != "comment" {
				typeLabel = utils.Dim(fmt.Sprintf(" [%s]", c.Type))
			}
			fmt.Printf("  %s  %s%s  %s\n", utils.Dim(fmt.Sprintf("#%d", c.ID)), utils.Dim(c.CreatedAt), typeLabel, c.Content)
		}

		return nil
	},
}

var commentDeleteCmd = &cobra.Command{
	Use:   "delete <comment-id>",
	Short: "Delete a comment",
	Args:  cobra.ExactArgs(1),
	RunE: func(cmd *cobra.Command, args []string) error {
		s := GetStore(cmd)

		id, err := strconv.ParseInt(args[0], 10, 64)
		if err != nil {
			return fmt.Errorf("invalid comment ID %q", args[0])
		}

		if err := s.DeleteComment(id); err != nil {
			return err
		}

		PrintOutput(cmd, fmt.Sprintf("deleted comment #%d", id), map[string]int64{"deleted": id})
		return nil
	},
}

func init() {
	AddJSONFlag(commentAddCmd)
	AddJSONFlag(commentListCmd)
	AddJSONFlag(commentDeleteCmd)
	commentCmd.AddCommand(commentAddCmd)
	commentCmd.AddCommand(commentListCmd)
	commentCmd.AddCommand(commentDeleteCmd)
	rootCmd.AddCommand(commentCmd)
}
