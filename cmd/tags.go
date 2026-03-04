package cmd

import (
	"fmt"

	"github.com/spf13/cobra"

	"vine/utils"
)

var tagsCmd = &cobra.Command{
	Use:   "tags",
	Short: "Manage tags",
}

var tagsListCmd = &cobra.Command{
	Use:   "list",
	Short: "List all tags with task counts",
	RunE: func(cmd *cobra.Command, args []string) error {
		s := GetStore(cmd)

		tags, err := s.ListTags()
		if err != nil {
			return err
		}

		if IsJSON(cmd) {
			PrintOutput(cmd, "", tags)
			return nil
		}

		if len(tags) == 0 {
			fmt.Println("No tags.")
			return nil
		}

		for _, t := range tags {
			count := utils.Dim(fmt.Sprintf("(%d)", t.Count))
			fmt.Printf("  %-20s %s\n", t.Name, count)
		}

		return nil
	},
}

var tagsPruneCmd = &cobra.Command{
	Use:   "prune",
	Short: "Remove tags with no associated tasks",
	RunE: func(cmd *cobra.Command, args []string) error {
		s := GetStore(cmd)

		n, err := s.PruneOrphanTags()
		if err != nil {
			return err
		}

		if IsJSON(cmd) {
			PrintOutput(cmd, "", map[string]int{"pruned": n})
			return nil
		}

		if n == 0 {
			fmt.Println("No orphan tags to prune.")
		} else {
			fmt.Printf("Pruned %d orphan tag(s).\n", n)
		}

		return nil
	},
}

func init() {
	AddJSONFlag(tagsListCmd)
	AddJSONFlag(tagsPruneCmd)
	tagsCmd.AddCommand(tagsListCmd)
	tagsCmd.AddCommand(tagsPruneCmd)
	rootCmd.AddCommand(tagsCmd)
}
