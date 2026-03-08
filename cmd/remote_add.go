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

By default, remotes use SSH tunneling for secure access. The vine server
on the remote host can bind to localhost — SSH handles authentication
and encryption using your existing SSH keys.

For direct HTTP access (less secure), use --http with optional --token and --tls.

Examples:
  vine remote add desktop 192.168.1.100                  # SSH tunnel (default)
  vine remote add desktop 192.168.1.100 --ssh-user royce # SSH with explicit user
  vine remote add cloud api.example.com --http --token s3cret --tls  # Direct HTTPS`,
	Args: cobra.ExactArgs(2),
	RunE: func(cmd *cobra.Command, args []string) error {
		name := args[0]
		host := args[1]
		port, _ := cmd.Flags().GetInt("port")
		token, _ := cmd.Flags().GetString("token")
		useTLS, _ := cmd.Flags().GetBool("tls")
		useHTTP, _ := cmd.Flags().GetBool("http")
		sshUser, _ := cmd.Flags().GetString("ssh-user")
		sshPort, _ := cmd.Flags().GetInt("ssh-port")

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

		if useHTTP {
			remote.Transport = "http"
		} else {
			remote.Transport = "ssh"
			remote.SSHUser = sshUser
			if sshPort != 22 {
				remote.SSHPort = sshPort
			}
		}

		if err := cfg.AddRemote(remote); err != nil {
			return err
		}

		if err := config.SaveRemotes(cfg); err != nil {
			return err
		}

		if remote.IsSSH() {
			target := host
			if sshUser != "" {
				target = sshUser + "@" + host
			}
			fmt.Printf("added remote %q (ssh %s, port %d)\n", name, target, port)
		} else {
			fmt.Printf("added remote %q (%s)\n", name, remote.URL())
		}
		return nil
	},
}

func init() {
	remoteAddCmd.Flags().Int("port", 7633, "vine server port on the remote host")
	remoteAddCmd.Flags().String("token", "", "bearer token for authentication")
	remoteAddCmd.Flags().Bool("tls", false, "use HTTPS (only with --http)")
	remoteAddCmd.Flags().Bool("http", false, "use direct HTTP instead of SSH tunnel")
	remoteAddCmd.Flags().String("ssh-user", "", "SSH username (default: current user)")
	remoteAddCmd.Flags().Int("ssh-port", 22, "SSH port")
	remoteCmd.AddCommand(remoteAddCmd)
}
