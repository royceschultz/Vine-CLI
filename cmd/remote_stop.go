package cmd

import (
	"fmt"
	"os"
	"path/filepath"
	"syscall"
	"time"

	"github.com/spf13/cobra"
)

var remoteStopCmd = &cobra.Command{
	Use:   "stop",
	Short: "Stop the server daemon",
	RunE: func(cmd *cobra.Command, args []string) error {
		dir, err := vineDir()
		if err != nil {
			return err
		}

		pidPath := filepath.Join(dir, "server.pid")
		pid, err := readPID(pidPath)
		if err != nil {
			return fmt.Errorf("server not running (no PID file)")
		}

		if !processExists(pid) {
			os.Remove(pidPath)
			return fmt.Errorf("server not running (stale PID file cleaned up)")
		}

		// Send SIGTERM for graceful shutdown.
		proc, err := os.FindProcess(pid)
		if err != nil {
			return fmt.Errorf("finding process %d: %w", pid, err)
		}

		if err := proc.Signal(syscall.SIGTERM); err != nil {
			return fmt.Errorf("sending SIGTERM to PID %d: %w", pid, err)
		}

		// Wait for process to exit (up to 5 seconds).
		for i := 0; i < 50; i++ {
			time.Sleep(100 * time.Millisecond)
			if !processExists(pid) {
				os.Remove(pidPath)
				fmt.Printf("server stopped (PID %d)\n", pid)
				return nil
			}
		}

		// Force kill if still running.
		proc.Signal(syscall.SIGKILL)
		os.Remove(pidPath)
		fmt.Printf("server killed (PID %d)\n", pid)
		return nil
	},
}

func init() {
	remoteCmd.AddCommand(remoteStopCmd)
}
