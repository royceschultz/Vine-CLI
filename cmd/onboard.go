package cmd

import (
	"fmt"
	"strings"

	"github.com/spf13/cobra"
)

const onboardText = `## Task Tracking

This project uses **vine** for task tracking.
Run ` + "`vine prime`" + ` for full workflow context at the start of each session.

**Quick reference:**
- ` + "`vine ready`" + ` — find tasks ready to work on
- ` + "`vine pick <id>`" + ` — claim a task (sets it to in_progress)
- ` + "`vine show <id>`" + ` — view task details, dependencies, and subtasks
- ` + "`vine create \"Title\" -t bug`" + ` — create a task (types: task, bug, feature, epic)
- ` + "`vine list`" + ` — list tasks (flags: -s status, -t type, --tag)
- ` + "`vine status`" + ` — project summary

Use ` + "`vine <command> --help`" + ` for full flag reference on any command.
Use ` + "`vine --json <command>`" + ` for machine-readable output.`

var onboardCmd = &cobra.Command{
	Use:   "onboard",
	Short: "Print onboarding snippet for AI agents",
	Long:  "Prints a quick-start guide for AI agents using vine for task tracking.",
	RunE: func(cmd *cobra.Command, args []string) error {
		raw, _ := cmd.Flags().GetBool("raw")

		if IsJSON(cmd) {
			PrintOutput(cmd, "", map[string]string{"content": onboardText})
			return nil
		}

		if raw {
			fmt.Println(onboardText)
			return nil
		}

		border := strings.Repeat("─", 60)
		fmt.Println(border)
		fmt.Println(onboardText)
		fmt.Println(border)

		return nil
	},
}

func init() {
	onboardCmd.Flags().Bool("raw", false, "print snippet without instructions or border")
	AddJSONFlag(onboardCmd)
	rootCmd.AddCommand(onboardCmd)
}
