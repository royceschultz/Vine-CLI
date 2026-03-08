package cmd

import (
	"fmt"
	"net"
	"strconv"
	"time"

	"github.com/spf13/cobra"

	"vine/utils"
)

var remoteStatusCmd = &cobra.Command{
	Use:   "status",
	Short: "Show server daemon status",
	RunE: func(cmd *cobra.Command, args []string) error {
		dir, err := vineDir()
		if err != nil {
			return err
		}

		cfg, _ := loadServerConfig(dir)

		pid, pidErr := readPID(dir + "/server.pid")
		alive := pidErr == nil && processExists(pid)

		// Determine address to probe.
		addr := "127.0.0.1:7633"
		if cfg != nil {
			addr = net.JoinHostPort(cfg.Bind, strconv.Itoa(cfg.Port))
		}

		if !alive {
			// Check for orphan via health endpoint.
			if health, err := probeHealth(addr); err == nil {
				fmt.Printf("  %s  %s\n", utils.Bold("status"), utils.StatusColor("cancelled").Sprint("orphan"))
				fmt.Printf("  %s  %d (PID file missing)\n", utils.Dim("pid:"), health.PID)
				fmt.Printf("  %s  %s\n", utils.Dim("addr:"), addr)
				fmt.Printf("  %s  %s\n", utils.Dim("started:"), health.StartedAt)
				return nil
			}
			fmt.Printf("  %s  %s\n", utils.Bold("status"), utils.Dim("stopped"))
			return nil
		}

		// Server is alive — probe health for details.
		health, healthErr := probeHealth(addr)

		fmt.Printf("  %s  %s\n", utils.Bold("status"), utils.StatusColor("done").Sprint("running"))
		fmt.Printf("  %s     %d\n", utils.Dim("pid:"), pid)
		fmt.Printf("  %s    %s\n", utils.Dim("addr:"), addr)

		if cfg != nil {
			if cfg.Token != "" {
				fmt.Printf("  %s    %s\n", utils.Dim("auth:"), "token")
			} else {
				fmt.Printf("  %s    %s\n", utils.Dim("auth:"), "none")
			}
			if cfg.TLSCert != "" {
				fmt.Printf("  %s     %s\n", utils.Dim("tls:"), "enabled")
			}
		}

		if healthErr == nil {
			started, err := time.Parse(time.RFC3339, health.StartedAt)
			if err == nil {
				uptime := time.Since(started).Round(time.Second)
				fmt.Printf("  %s  %s\n", utils.Dim("uptime:"), uptime)
			}
		}

		return nil
	},
}

func init() {
	remoteCmd.AddCommand(remoteStatusCmd)
}
