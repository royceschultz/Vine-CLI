package cmd

import (
	"fmt"
	"os"

	"github.com/spf13/cobra"
)

var migrateCmd = &cobra.Command{
	Use:   "migrate",
	Short: "Move or merge databases between local and global storage",
	Run: func(cmd *cobra.Command, args []string) {
		fmt.Fprintln(os.Stderr, "not yet implemented")
		os.Exit(1)
	},
}

func init() {
	AddJSONFlag(migrateCmd)
	rootCmd.AddCommand(migrateCmd)
}
