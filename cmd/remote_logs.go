package cmd

import (
	"bufio"
	"fmt"
	"io"
	"os"
	"path/filepath"
	"time"

	"github.com/spf13/cobra"
)

var remoteLogsCmd = &cobra.Command{
	Use:   "logs",
	Short: "View server logs",
	Long: `View the vine server log file (~/.vine/server.log).

Examples:
  vine remote logs              # last 50 lines
  vine remote logs --tail 100   # last 100 lines
  vine remote logs --follow     # stream new lines (like tail -f)`,
	RunE: func(cmd *cobra.Command, args []string) error {
		dir, err := vineDir()
		if err != nil {
			return err
		}

		logPath := filepath.Join(dir, "server.log")
		tail, _ := cmd.Flags().GetInt("tail")
		follow, _ := cmd.Flags().GetBool("follow")

		if _, err := os.Stat(logPath); os.IsNotExist(err) {
			return fmt.Errorf("no log file found at %s", logPath)
		}

		// Print last N lines.
		if err := printTail(logPath, tail); err != nil {
			return err
		}

		if !follow {
			return nil
		}

		// Follow mode: watch for new lines.
		return followLog(logPath)
	},
}

func init() {
	remoteLogsCmd.Flags().IntP("tail", "n", 50, "number of lines to show")
	remoteLogsCmd.Flags().BoolP("follow", "f", false, "follow log output")
	remoteCmd.AddCommand(remoteLogsCmd)
}

// printTail prints the last n lines of a file.
func printTail(path string, n int) error {
	f, err := os.Open(path)
	if err != nil {
		return err
	}
	defer f.Close()

	// Read all lines (simple approach — log files are bounded by rotation).
	var lines []string
	scanner := bufio.NewScanner(f)
	for scanner.Scan() {
		lines = append(lines, scanner.Text())
	}
	if err := scanner.Err(); err != nil {
		return err
	}

	start := 0
	if len(lines) > n {
		start = len(lines) - n
	}

	for _, line := range lines[start:] {
		fmt.Println(line)
	}
	return nil
}

// followLog tails a file, printing new lines as they appear.
func followLog(path string) error {
	f, err := os.Open(path)
	if err != nil {
		return err
	}
	defer f.Close()

	// Seek to end.
	if _, err := f.Seek(0, io.SeekEnd); err != nil {
		return err
	}

	reader := bufio.NewReader(f)
	for {
		line, err := reader.ReadString('\n')
		if err != nil {
			// No new data — wait and retry.
			time.Sleep(200 * time.Millisecond)
			continue
		}
		fmt.Print(line)
	}
}
