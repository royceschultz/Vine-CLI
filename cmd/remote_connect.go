package cmd

import (
	"fmt"

	"github.com/spf13/cobra"

	"vine/client"
	"vine/config"
	"vine/utils"
)

var remoteConnectCmd = &cobra.Command{
	Use:   "connect <name>",
	Short: "Open a persistent SSH tunnel to a remote",
	Long: `Establish a persistent SSH tunnel to a remote vine server.

The tunnel runs in the background and is reused by subsequent vine commands.
Use 'vine remote disconnect' to tear it down.

If a tunnel is already active, this is a no-op.`,
	Args: cobra.ExactArgs(1),
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

		if !remote.IsSSH() {
			return fmt.Errorf("remote %q uses direct HTTP — tunnels are only for SSH remotes", name)
		}

		// Check if already connected.
		existing, _ := client.LoadTunnel(name)
		if client.IsTunnelAlive(existing) {
			fmt.Printf("  %s  %s  %s\n",
				utils.Bold(name),
				utils.StatusColor("done").Sprint("already connected"),
				utils.Dim(fmt.Sprintf("pid=%d port=%d", existing.PID, existing.LocalPort)),
			)
			return nil
		}

		info, err := client.Connect(remote)
		if err != nil {
			fmt.Printf("  %s  %s\n",
				utils.Bold(name),
				utils.StatusColor("cancelled").Sprint("connection failed"),
			)
			return err
		}

		fmt.Printf("  %s  %s  %s\n",
			utils.Bold(name),
			utils.StatusColor("done").Sprint("connected"),
			utils.Dim(fmt.Sprintf("pid=%d port=%d", info.PID, info.LocalPort)),
		)
		return nil
	},
}

func init() {
	remoteCmd.AddCommand(remoteConnectCmd)
}
