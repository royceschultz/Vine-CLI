package cmd

import (
	"fmt"

	"github.com/spf13/cobra"

	"vine/config"
	"vine/utils"
)

var remoteListCmd = &cobra.Command{
	Use:     "list",
	Aliases: []string{"ls"},
	Short:   "List configured remotes",
	RunE: func(cmd *cobra.Command, args []string) error {
		cfg, err := config.LoadRemotes()
		if err != nil {
			return err
		}

		if IsJSON(cmd) {
			PrintOutput(cmd, "", cfg.Remotes)
			return nil
		}

		if len(cfg.Remotes) == 0 {
			fmt.Println("No remotes configured. Use 'vine remote add' to add one.")
			return nil
		}

		for _, r := range cfg.Remotes {
			tokenStatus := utils.Dim("no auth")
			if r.Token != "" {
				tokenStatus = utils.Dim("token")
			}

			scheme := "http"
			if r.TLS {
				scheme = "https"
			}

			fmt.Printf("  %s  %s://%s:%d  %s\n",
				utils.Bold(r.Name),
				scheme, r.Host, r.Port,
				tokenStatus,
			)
		}

		return nil
	},
}

func init() {
	AddJSONFlag(remoteListCmd)
	remoteCmd.AddCommand(remoteListCmd)
}
