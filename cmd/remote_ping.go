package cmd

import (
	"crypto/tls"
	"encoding/json"
	"fmt"
	"net/http"
	"time"

	"github.com/spf13/cobra"

	"vine/config"
	"vine/server"
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

		url := remote.URL() + "/api/health"

		client := &http.Client{Timeout: 5 * time.Second}
		if remote.TLS {
			client.Transport = &http.Transport{
				TLSClientConfig: &tls.Config{},
			}
		}

		start := time.Now()
		resp, err := client.Get(url)
		latency := time.Since(start)

		if err != nil {
			fmt.Printf("  %s  %s  %s\n",
				utils.Bold(name),
				utils.StatusColor("cancelled").Sprint("unreachable"),
				utils.Dim(err.Error()),
			)
			return fmt.Errorf("cannot reach %s", name)
		}
		defer resp.Body.Close()

		var health server.HealthResponse
		if err := json.NewDecoder(resp.Body).Decode(&health); err != nil || health.Service != "vine" {
			fmt.Printf("  %s  %s  %s\n",
				utils.Bold(name),
				utils.StatusColor("cancelled").Sprint("not a vine server"),
				utils.Dim(url),
			)
			return fmt.Errorf("endpoint is not a vine server")
		}

		fmt.Printf("  %s  %s  %s  %s\n",
			utils.Bold(name),
			utils.StatusColor("done").Sprint("ok"),
			utils.Dim(fmt.Sprintf("pid=%d", health.PID)),
			utils.Dim(fmt.Sprintf("%s", latency.Round(time.Microsecond))),
		)

		return nil
	},
}

func init() {
	remoteCmd.AddCommand(remotePingCmd)
}
