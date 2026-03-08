package cmd

import (
	"fmt"
	"time"

	"github.com/spf13/cobra"

	"vine/client"
	"vine/config"
	"vine/utils"
)

var remotePingCmd = &cobra.Command{
	Use:   "ping <name>",
	Short: "Test connectivity to a remote vine server",
	Args:  cobra.ExactArgs(1),
	RunE: func(cmd *cobra.Command, args []string) error {
		name := args[0]

		cfg, err := config.LoadRemotes()
		if err != nil {
			return err
		}

		remote := cfg.GetRemote(name)
		if remote == nil {
			return fmt.Errorf("remote %q not found. Use 'vine remote list' to see configured remotes", name)
		}

		c, err := client.New(remote)
		if err != nil {
			fmt.Printf("  %s  %s  %s\n",
				utils.Bold(name),
				utils.StatusColor("cancelled").Sprint("unreachable"),
				utils.Dim(err.Error()),
			)
			return fmt.Errorf("cannot reach %s", name)
		}
		defer c.Close()

		start := time.Now()
		health, err := c.Health()
		latency := time.Since(start)

		if err != nil {
			fmt.Printf("  %s  %s  %s\n",
				utils.Bold(name),
				utils.StatusColor("cancelled").Sprint("unreachable"),
				utils.Dim(err.Error()),
			)
			return fmt.Errorf("cannot reach %s", name)
		}

		transport := "ssh"
		if !remote.IsSSH() {
			transport = remote.Scheme()
		}

		fmt.Printf("  %s  %s  %s  %s  %s\n",
			utils.Bold(name),
			utils.StatusColor("done").Sprint("ok"),
			utils.Dim(transport),
			utils.Dim(fmt.Sprintf("pid=%d", health.PID)),
			utils.Dim(fmt.Sprintf("%s", latency.Round(time.Microsecond))),
		)

		return nil
	},
}

func init() {
	remoteCmd.AddCommand(remotePingCmd)
}
