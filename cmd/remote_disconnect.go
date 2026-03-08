package cmd

import (
	"fmt"

	"github.com/spf13/cobra"

	"vine/client"
	"vine/utils"
)

var remoteDisconnectCmd = &cobra.Command{
	Use:   "disconnect <name>",
	Short: "Close a persistent SSH tunnel to a remote",
	Args:  cobra.ExactArgs(1),
	RunE: func(cmd *cobra.Command, args []string) error {
		name := args[0]

		if err := client.Disconnect(name); err != nil {
			return err
		}

		fmt.Printf("  %s  %s\n",
			utils.Bold(name),
			utils.Dim("disconnected"),
		)
		return nil
	},
}

var remoteDisconnectAllCmd = &cobra.Command{
	Use:   "disconnect-all",
	Short: "Close all persistent SSH tunnels",
	RunE: func(cmd *cobra.Command, args []string) error {
		tunnels, err := client.ListTunnels()
		if err != nil {
			return err
		}

		if len(tunnels) == 0 {
			fmt.Println("No active tunnels.")
			return nil
		}

		for _, t := range tunnels {
			if err := client.Disconnect(t.Name); err != nil {
				fmt.Printf("  %s  %s  %s\n",
					utils.Bold(t.Name),
					utils.StatusColor("cancelled").Sprint("failed"),
					utils.Dim(err.Error()),
				)
			} else {
				fmt.Printf("  %s  %s\n",
					utils.Bold(t.Name),
					utils.Dim("disconnected"),
				)
			}
		}
		return nil
	},
}

func init() {
	remoteCmd.AddCommand(remoteDisconnectCmd)
	remoteCmd.AddCommand(remoteDisconnectAllCmd)
}
