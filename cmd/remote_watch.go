package cmd

import (
	"encoding/json"
	"fmt"
	"os"
	"os/signal"
	"syscall"
	"time"

	"github.com/gorilla/websocket"
	"github.com/spf13/cobra"

	"vine/client"
	"vine/config"
	"vine/server"
	"vine/utils"
)

var remoteWatchCmd = &cobra.Command{
	Use:   "watch <name>",
	Short: "Watch a remote for changes",
	Long: `Connect to a remote vine server's WebSocket endpoint and print change
events as they arrive. Watches all projects by default, or a specific
project with --project. Automatically reconnects on disconnect.

Examples:
  vine remote watch desktop                  # watch all projects
  vine remote watch desktop --project myapp  # watch one project
  vine remote watch desktop --json`,
	Args: cobra.ExactArgs(1),
	RunE: func(cmd *cobra.Command, args []string) error {
		name := args[0]
		project, _ := cmd.Flags().GetString("project")

		cfg, err := config.LoadRemotes()
		if err != nil {
			return err
		}

		remote := cfg.GetRemote(name)
		if remote == nil {
			return fmt.Errorf("remote %q not found", name)
		}

		// Resolve the WebSocket URL.
		wsURL, err := resolveWSURL(remote, project)
		if err != nil {
			return err
		}

		// Handle Ctrl+C.
		sigCh := make(chan os.Signal, 1)
		signal.Notify(sigCh, syscall.SIGINT, syscall.SIGTERM)

		label := name
		if project != "" {
			label = name + "/" + project
		}
		fmt.Fprintf(os.Stderr, "watching %s for changes (ctrl+c to stop)\n", label)

		jsonOutput := IsJSON(cmd)

		for {
			err := watchLoop(wsURL, jsonOutput, sigCh)
			if err == errInterrupted {
				fmt.Fprintln(os.Stderr)
				return nil
			}
			if err != nil {
				fmt.Fprintf(os.Stderr, "disconnected: %s\n", err)
			}

			// Reconnect after a brief pause.
			fmt.Fprintf(os.Stderr, "reconnecting in 3s...\n")
			select {
			case <-sigCh:
				return nil
			case <-time.After(3 * time.Second):
			}
		}
	},
}

var errInterrupted = fmt.Errorf("interrupted")

func watchLoop(url string, jsonOutput bool, sigCh chan os.Signal) error {
	conn, _, err := websocket.DefaultDialer.Dial(url, nil)
	if err != nil {
		return fmt.Errorf("connecting: %w", err)
	}
	defer conn.Close()

	if !jsonOutput {
		fmt.Fprintf(os.Stderr, "connected\n")
	}

	msgCh := make(chan []byte)
	errCh := make(chan error, 1)

	go func() {
		for {
			_, msg, err := conn.ReadMessage()
			if err != nil {
				errCh <- err
				return
			}
			msgCh <- msg
		}
	}()

	for {
		select {
		case <-sigCh:
			conn.WriteMessage(websocket.CloseMessage,
				websocket.FormatCloseMessage(websocket.CloseNormalClosure, ""))
			return errInterrupted
		case msg := <-msgCh:
			if jsonOutput {
				fmt.Println(string(msg))
			} else {
				printWatchEvent(msg)
			}
		case err := <-errCh:
			return err
		}
	}
}

func printWatchEvent(data []byte) {
	var event server.WatchEvent
	// Best effort parse — print raw if it fails.
	if err := json.Unmarshal(data, &event); err != nil {
		fmt.Println(string(data))
		return
	}

	t, err := time.Parse(time.RFC3339, event.Timestamp)
	timestamp := event.Timestamp
	if err == nil {
		timestamp = t.Local().Format("15:04:05")
	}

	fmt.Printf("  %s  %s  %s\n",
		utils.Dim(timestamp),
		utils.Bold(event.Project),
		event.Type,
	)
}

func resolveWSURL(remote *config.Remote, project string) (string, error) {
	path := "/api/watch"
	if project != "" {
		path = fmt.Sprintf("/api/projects/%s/watch", project)
	}

	if remote.IsSSH() {
		info, err := client.Connect(remote)
		if err != nil {
			return "", fmt.Errorf("connecting to %s: %w", remote.Name, err)
		}
		return fmt.Sprintf("ws://127.0.0.1:%d%s", info.LocalPort, path), nil
	}

	scheme := "ws"
	if remote.TLS {
		scheme = "wss"
	}
	return fmt.Sprintf("%s://%s:%d%s", scheme, remote.Host, remote.Port, path), nil
}

func init() {
	remoteWatchCmd.Flags().String("project", "", "project to watch")
	AddJSONFlag(remoteWatchCmd)
	remoteCmd.AddCommand(remoteWatchCmd)
}
