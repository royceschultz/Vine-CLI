package cmd

import (
	"context"
	"crypto/tls"
	"encoding/json"
	"fmt"
	"log"
	"net"
	"net/http"
	"os"
	"os/signal"
	"path/filepath"
	"strconv"
	"syscall"
	"time"

	"github.com/spf13/cobra"
	"github.com/spf13/pflag"

	"vine/server"
)

var serveCmd = &cobra.Command{
	Use:   "serve",
	Short: "Start the vine HTTP API server",
	Long: `Start a persistent HTTP server that exposes read-only vine data.

By default, binds to 127.0.0.1:7633 (localhost only, suitable for SSH tunnels).
Use --bind 0.0.0.0 to listen on all interfaces (pair with --token for auth).

Examples:
  vine remote serve                          # localhost:7633, no auth
  vine remote serve --port 8080              # localhost:8080
  vine remote serve --bind 0.0.0.0 --token s3cret  # all interfaces, token auth
  vine remote serve --tls-cert cert.pem --tls-key key.pem  # HTTPS`,
	RunE: runServe,
}

func init() {
	serveCmd.Flags().Int("port", 7633, "port to listen on")
	serveCmd.Flags().String("bind", "127.0.0.1", "address to bind to")
	serveCmd.Flags().String("token", "", "bearer token for authentication (optional)")
	serveCmd.Flags().String("tls-cert", "", "path to TLS certificate file (optional)")
	serveCmd.Flags().String("tls-key", "", "path to TLS private key file (optional)")
	serveCmd.Flags().Bool("foreground", false, "run in foreground instead of daemonizing")
	remoteCmd.AddCommand(serveCmd)
}

func vineDir() (string, error) {
	home, err := os.UserHomeDir()
	if err != nil {
		return "", fmt.Errorf("resolving home directory: %w", err)
	}
	dir := filepath.Join(home, ".vine")
	if err := os.MkdirAll(dir, 0o755); err != nil {
		return "", fmt.Errorf("creating ~/.vine: %w", err)
	}
	return dir, nil
}

func runServe(cmd *cobra.Command, args []string) error {
	port, _ := cmd.Flags().GetInt("port")
	bind, _ := cmd.Flags().GetString("bind")
	token, _ := cmd.Flags().GetString("token")
	tlsCert, _ := cmd.Flags().GetString("tls-cert")
	tlsKey, _ := cmd.Flags().GetString("tls-key")
	foreground, _ := cmd.Flags().GetBool("foreground")

	// Validate TLS flags come in pairs.
	if (tlsCert == "") != (tlsKey == "") {
		return fmt.Errorf("--tls-cert and --tls-key must be provided together")
	}

	dir, err := vineDir()
	if err != nil {
		return err
	}

	pidPath := filepath.Join(dir, "server.pid")
	addr := net.JoinHostPort(bind, strconv.Itoa(port))

	// Check if already running via PID file.
	if pid, err := readPID(pidPath); err == nil {
		if processExists(pid) && isVineServer(addr, pid) {
			return fmt.Errorf("server already running (PID %d). Use 'vine remote stop' to stop it", pid)
		}
		// Stale PID file or PID reuse — clean up.
		os.Remove(pidPath)
	}

	// Check for orphan server (no PID file, but something is on our port).
	if health, err := probeHealth(addr); err == nil {
		return fmt.Errorf("vine server already running on %s (PID %d, started %s) but PID file is missing.\nUse 'kill %d' to stop it, or choose a different --port",
			addr, health.PID, health.StartedAt, health.PID)
	}

	if !foreground {
		return daemonize(cmd, dir)
	}

	return serveForeground(dir, pidPath, bind, port, token, tlsCert, tlsKey)
}

// serverConfig is persisted to ~/.vine/server.json so restart can use it.
type serverConfig struct {
	Port    int    `json:"port"`
	Bind    string `json:"bind"`
	Token   string `json:"token,omitempty"`
	TLSCert string `json:"tls_cert,omitempty"`
	TLSKey  string `json:"tls_key,omitempty"`
}

func saveServerConfig(dir string, cfg serverConfig) error {
	data, _ := json.MarshalIndent(cfg, "", "  ")
	return os.WriteFile(filepath.Join(dir, "server.json"), append(data, '\n'), 0o644)
}

func loadServerConfig(dir string) (*serverConfig, error) {
	data, err := os.ReadFile(filepath.Join(dir, "server.json"))
	if err != nil {
		return nil, err
	}
	var cfg serverConfig
	if err := json.Unmarshal(data, &cfg); err != nil {
		return nil, err
	}
	return &cfg, nil
}

func serveForeground(dir, pidPath, bind string, port int, token, tlsCert, tlsKey string) error {
	// Save config for restart.
	saveServerConfig(dir, serverConfig{
		Port:    port,
		Bind:    bind,
		Token:   token,
		TLSCert: tlsCert,
		TLSKey:  tlsKey,
	})

	// Set up log file.
	logPath := filepath.Join(dir, "server.log")
	logFile, err := server.NewRotatingLog(logPath, 0, 0)
	if err != nil {
		return fmt.Errorf("opening log file: %w", err)
	}
	defer logFile.Close()

	logger := log.New(logFile, "", log.LstdFlags)

	// Build handler chain.
	srv := server.New()

	// Attach file watcher for WebSocket notifications.
	// The watcher only starts monitoring when clients connect.
	watcher := server.NewWatcher(logger)
	srv.SetWatcher(watcher)

	var handler http.Handler = srv.Handler()
	handler = server.RequestLogger(logger)(handler)
	handler = server.TokenAuth(token)(handler)

	addr := net.JoinHostPort(bind, strconv.Itoa(port))

	httpServer := &http.Server{
		Addr:    addr,
		Handler: handler,
	}

	if tlsCert != "" {
		cert, err := tls.LoadX509KeyPair(tlsCert, tlsKey)
		if err != nil {
			return fmt.Errorf("loading TLS certificate: %w", err)
		}
		httpServer.TLSConfig = &tls.Config{
			Certificates: []tls.Certificate{cert},
		}
	}

	// Write PID file.
	if err := writePID(pidPath, os.Getpid()); err != nil {
		return fmt.Errorf("writing PID file: %w", err)
	}
	defer os.Remove(pidPath)

	// Listen.
	ln, err := net.Listen("tcp", addr)
	if err != nil {
		return fmt.Errorf("listening on %s: %w", addr, err)
	}

	scheme := "http"
	if tlsCert != "" {
		scheme = "https"
	}

	logger.Printf("vine server started on %s://%s (PID %d)", scheme, addr, os.Getpid())
	fmt.Fprintf(os.Stderr, "vine server listening on %s://%s\n", scheme, addr)

	// Graceful shutdown on signal.
	ctx, stop := signal.NotifyContext(context.Background(), syscall.SIGINT, syscall.SIGTERM)
	defer stop()

	errCh := make(chan error, 1)
	go func() {
		if tlsCert != "" {
			errCh <- httpServer.ServeTLS(ln, "", "")
		} else {
			errCh <- httpServer.Serve(ln)
		}
	}()

	select {
	case <-ctx.Done():
		logger.Printf("shutting down...")
		watcher.Stop()
		shutdownCtx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
		defer cancel()
		return httpServer.Shutdown(shutdownCtx)
	case err := <-errCh:
		if err != nil && err != http.ErrServerClosed {
			return err
		}
		return nil
	}
}

// daemonize re-runs the serve command in the background with --foreground.
func daemonize(cmd *cobra.Command, dir string) error {
	exe, err := os.Executable()
	if err != nil {
		return fmt.Errorf("finding executable: %w", err)
	}

	// Reconstruct args with --foreground added.
	args := []string{exe, "remote", "serve", "--foreground"}

	// Forward flags.
	cmd.Flags().Visit(func(f *pflag.Flag) {
		if f.Name == "foreground" {
			return
		}
		args = append(args, fmt.Sprintf("--%s=%s", f.Name, f.Value.String()))
	})

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

	proc, err := os.StartProcess(exe, args, attr)
	if err != nil {
		logFile.Close()
		return fmt.Errorf("starting daemon: %w", err)
	}
	logFile.Close()

	// Detach from the child process.
	proc.Release()

	// Wait briefly to check if it started OK.
	time.Sleep(200 * time.Millisecond)

	pidPath := filepath.Join(dir, "server.pid")
	if pid, err := readPID(pidPath); err == nil {
		fmt.Fprintf(os.Stderr, "vine server started (PID %d)\n", pid)
	} else {
		fmt.Fprintln(os.Stderr, "vine server started in background")
	}

	return nil
}

func writePID(path string, pid int) error {
	return os.WriteFile(path, []byte(strconv.Itoa(pid)+"\n"), 0o644)
}

func readPID(path string) (int, error) {
	data, err := os.ReadFile(path)
	if err != nil {
		return 0, err
	}
	s := string(data)
	// Trim whitespace/newline.
	for len(s) > 0 && (s[len(s)-1] == '\n' || s[len(s)-1] == '\r' || s[len(s)-1] == ' ') {
		s = s[:len(s)-1]
	}
	return strconv.Atoi(s)
}

func processExists(pid int) bool {
	proc, err := os.FindProcess(pid)
	if err != nil {
		return false
	}
	// Signal 0 checks if process exists without sending a signal.
	err = proc.Signal(syscall.Signal(0))
	return err == nil
}

// probeHealth hits the /api/health endpoint to check if a vine server is listening.
func probeHealth(addr string) (*server.HealthResponse, error) {
	client := &http.Client{Timeout: 2 * time.Second}
	resp, err := client.Get(fmt.Sprintf("http://%s/api/health", addr))
	if err != nil {
		return nil, err
	}
	defer resp.Body.Close()

	var health server.HealthResponse
	if err := json.NewDecoder(resp.Body).Decode(&health); err != nil {
		return nil, err
	}
	if health.Service != "vine" {
		return nil, fmt.Errorf("not a vine server")
	}
	return &health, nil
}

// isVineServer checks whether the process at the given PID is actually a vine server
// by probing the health endpoint. This mitigates PID reuse — if the OS recycled the PID
// for an unrelated process, the health check will fail and we'll treat it as stale.
func isVineServer(addr string, expectedPID int) bool {
	health, err := probeHealth(addr)
	if err != nil {
		// Can't reach the server — might be starting up, or PID was reused.
		// Fall back to process existence check only.
		return true
	}
	return health.PID == expectedPID
}
