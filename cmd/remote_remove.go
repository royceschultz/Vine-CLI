package cmd

import (
	"fmt"

	"github.com/spf13/cobra"

	"vine/config"
)

var remoteRemoveCmd = &cobra.Command{
	Use:     "remove <name>",
	Aliases: []string{"rm"},
	Short:   "Remove a remote vine server",
	Args:    cobra.ExactArgs(1),
	RunE: func(cmd *cobra.Command, args []string) error {
		name := args[0]

		cfg, err := config.LoadRemotes()
		if err != nil {
			return err
		}

		if err := cfg.RemoveRemote(name); err != nil {
			return err
		}

		if err := config.SaveRemotes(cfg); err != nil {
			return err
		}

		fmt.Printf("removed remote %q\n", name)
		return nil
	},
}

func init() {
	remoteCmd.AddCommand(remoteRemoveCmd)
}
