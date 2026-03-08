package cmd

import (
	"github.com/spf13/cobra"
)

var remoteCmd = &cobra.Command{
	Use:   "remote",
	Short: "Manage remote vine servers",
	Long:  "Commands for serving vine data over the network and connecting to remote vine instances.",
}

func init() {
	rootCmd.AddCommand(remoteCmd)
}
