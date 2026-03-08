package cmd

import (
	"fmt"

	"github.com/spf13/cobra"

	"vine/client"
	"vine/config"
)

var remoteProjectsCmd = &cobra.Command{
	Use:   "projects <name>",
	Short: "List projects on a remote vine server",
	Args:  cobra.ExactArgs(1),
	RunE: func(cmd *cobra.Command, args []string) error {
		name := args[0]

		cfg, err := config.LoadRemotes()
		if err != nil {
			return err
		}

		remote := cfg.GetRemote(name)
		if remote == nil {
			return fmt.Errorf("remote %q not found", name)
		}

		c, err := client.New(remote)
		if err != nil {
			return fmt.Errorf("connecting to %s: %w", name, err)
		}
		defer c.Close()
		projects, err := c.ListProjects()
		if err != nil {
			return fmt.Errorf("querying %s: %w", name, err)
		}

		if IsJSON(cmd) {
			PrintOutput(cmd, "", projects)
			return nil
		}

		if len(projects) == 0 {
			fmt.Printf("No projects on %s.\n", name)
			return nil
		}

		fmt.Printf("%d projects on %s:\n\n", len(projects), name)
		for _, p := range projects {
			fmt.Printf("  %s\n", p)
		}

		return nil
	},
}

func init() {
	AddJSONFlag(remoteProjectsCmd)
	remoteCmd.AddCommand(remoteProjectsCmd)
}
