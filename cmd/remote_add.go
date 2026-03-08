package cmd

import (
	"fmt"

	"github.com/spf13/cobra"

	"vine/config"
)

var remoteAddCmd = &cobra.Command{
	Use:   "add <name> <host>",
	Short: "Add a remote vine server",
	Long: `Add a remote vine server connection.

Examples:
  vine remote add work 192.168.1.100
  vine remote add work myhost.local --port 8080
  vine remote add work myhost.local --token s3cret --tls`,
	Args: cobra.ExactArgs(2),
	RunE: func(cmd *cobra.Command, args []string) error {
		name := args[0]
		host := args[1]
		port, _ := cmd.Flags().GetInt("port")
		token, _ := cmd.Flags().GetString("token")
		useTLS, _ := cmd.Flags().GetBool("tls")

		cfg, err := config.LoadRemotes()
		if err != nil {
			return err
		}

		remote := config.Remote{
			Name:  name,
			Host:  host,
			Port:  port,
			Token: token,
			TLS:   useTLS,
		}

		if err := cfg.AddRemote(remote); err != nil {
			return err
		}

		if err := config.SaveRemotes(cfg); err != nil {
			return err
		}

		fmt.Printf("added remote %q (%s)\n", name, remote.URL())
		return nil
	},
}

func init() {
	remoteAddCmd.Flags().Int("port", 7633, "server port")
	remoteAddCmd.Flags().String("token", "", "bearer token for authentication")
	remoteAddCmd.Flags().Bool("tls", false, "use HTTPS")
	remoteCmd.AddCommand(remoteAddCmd)
}
