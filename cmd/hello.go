package cmd

import "github.com/spf13/cobra"

var helloCmd = &cobra.Command{
	Use:   "hello",
	Short: "Say hello from vine",
	Run: func(cmd *cobra.Command, args []string) {
		PrintOutput(cmd, "hello from vine", map[string]string{
			"message": "hello from vine",
		})
	},
}

func init() {
	AddJSONFlag(helloCmd)
	rootCmd.AddCommand(helloCmd)
}
