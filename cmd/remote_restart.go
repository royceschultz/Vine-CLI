package cmd

import (
	"fmt"
	"os"
	"path/filepath"
	"strconv"
	"syscall"
	"time"

	"github.com/spf13/cobra"
)

var remoteRestartCmd = &cobra.Command{
	Use:   "restart",
	Short: "Restart the server daemon",
	Long:  "Stop the server and start it again with the same configuration.",
	RunE: func(cmd *cobra.Command, args []string) error {
		dir, err := vineDir()
		if err != nil {
			return err
		}

		// Load saved config.
		cfg, err := loadServerConfig(dir)
		if err != nil {
			return fmt.Errorf("no server config found (has the server been started before?)")
		}

		// Stop if running.
		pidPath := filepath.Join(dir, "server.pid")
		if pid, err := readPID(pidPath); err == nil && processExists(pid) {
			proc, _ := os.FindProcess(pid)
			proc.Signal(syscall.SIGTERM)

			for i := 0; i < 50; i++ {
				time.Sleep(100 * time.Millisecond)
				if !processExists(pid) {
					break
				}
			}
			if processExists(pid) {
				proc.Signal(syscall.SIGKILL)
				time.Sleep(100 * time.Millisecond)
			}
			os.Remove(pidPath)
			fmt.Printf("stopped (PID %d)\n", pid)
		}

		// Spawn a new daemon with saved config using os.StartProcess.
		exe, err := os.Executable()
		if err != nil {
			return fmt.Errorf("finding executable: %w", err)
		}

		execArgs := []string{exe, "remote", "serve", "--foreground",
			"--port", strconv.Itoa(cfg.Port),
			"--bind", cfg.Bind,
		}
		if cfg.Token != "" {
			execArgs = append(execArgs, "--token", cfg.Token)
		}
		if cfg.TLSCert != "" {
			execArgs = append(execArgs, "--tls-cert", cfg.TLSCert, "--tls-key", cfg.TLSKey)
		}

		logPath := filepath.Join(dir, "server.log")
		logFile, err := os.OpenFile(logPath, os.O_CREATE|os.O_WRONLY|os.O_APPEND, 0o644)
		if err != nil {
			return fmt.Errorf("opening log file: %w", err)
		}

		attr := &os.ProcAttr{
			Dir:   "/",
			Env:   os.Environ(),
			Files: []*os.File{os.Stdin, logFile, logFile},
			Sys: &syscall.SysProcAttr{
				Setsid: true,
			},
		}

		proc, err := os.StartProcess(exe, execArgs, attr)
		if err != nil {
			logFile.Close()
			return fmt.Errorf("starting daemon: %w", err)
		}
		logFile.Close()
		proc.Release()

		time.Sleep(200 * time.Millisecond)

		if pid, err := readPID(pidPath); err == nil {
			fmt.Printf("started (PID %d)\n", pid)
		} else {
			fmt.Println("server started in background")
		}

		return nil
	},
}

func init() {
	remoteCmd.AddCommand(remoteRestartCmd)
}
