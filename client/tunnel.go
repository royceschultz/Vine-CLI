package client

import (
	"encoding/json"
	"fmt"
	"net"
	"os"
	"os/exec"
	"path/filepath"
	"strconv"
	"syscall"
	"time"

	"vine/config"
)

// TunnelInfo describes a persistent SSH tunnel.
type TunnelInfo struct {
	Name      string `json:"name"`
	PID       int    `json:"pid"`
	LocalPort int    `json:"local_port"`
	Host      string `json:"host"`
	SSHUser   string `json:"ssh_user,omitempty"`
	SSHPort   int    `json:"ssh_port"`
	VinePort  int    `json:"vine_port"`
}

func tunnelsDir() (string, error) {
	home, err := os.UserHomeDir()
	if err != nil {
		return "", err
	}
	dir := filepath.Join(home, ".vine", "tunnels")
	if err := os.MkdirAll(dir, 0o755); err != nil {
		return "", err
	}
	return dir, nil
}

func tunnelPath(name string) (string, error) {
	dir, err := tunnelsDir()
	if err != nil {
		return "", err
	}
	return filepath.Join(dir, name+".json"), nil
}

// LoadTunnel reads the tunnel info for a named remote. Returns nil if no tunnel file exists.
func LoadTunnel(name string) (*TunnelInfo, error) {
	path, err := tunnelPath(name)
	if err != nil {
		return nil, err
	}
	data, err := os.ReadFile(path)
	if err != nil {
		if os.IsNotExist(err) {
			return nil, nil
		}
		return nil, err
	}
	var info TunnelInfo
	if err := json.Unmarshal(data, &info); err != nil {
		return nil, err
	}
	return &info, nil
}

func saveTunnel(info *TunnelInfo) error {
	path, err := tunnelPath(info.Name)
	if err != nil {
		return err
	}
	data, _ := json.MarshalIndent(info, "", "  ")
	return os.WriteFile(path, append(data, '\n'), 0o644)
}

func removeTunnelFile(name string) {
	if path, err := tunnelPath(name); err == nil {
		os.Remove(path)
	}
}

// IsTunnelAlive checks if a persistent tunnel is still running and reachable.
func IsTunnelAlive(info *TunnelInfo) bool {
	if info == nil {
		return false
	}
	// Check process exists.
	proc, err := os.FindProcess(info.PID)
	if err != nil {
		return false
	}
	if err := proc.Signal(syscall.Signal(0)); err != nil {
		return false
	}
	// Check the local port is accepting connections.
	conn, err := net.DialTimeout("tcp", fmt.Sprintf("127.0.0.1:%d", info.LocalPort), 2*time.Second)
	if err != nil {
		return false
	}
	conn.Close()
	return true
}

// Connect establishes a persistent SSH tunnel for the named remote.
// If a tunnel is already alive, it returns the existing tunnel info.
func Connect(remote *config.Remote) (*TunnelInfo, error) {
	// Check for existing tunnel.
	existing, err := LoadTunnel(remote.Name)
	if err != nil {
		return nil, fmt.Errorf("checking existing tunnel: %w", err)
	}
	if IsTunnelAlive(existing) {
		return existing, nil
	}
	// Clean up stale tunnel file if present.
	if existing != nil {
		removeTunnelFile(remote.Name)
	}

	// Find a free local port.
	localPort, err := freePort()
	if err != nil {
		return nil, fmt.Errorf("finding free port: %w", err)
	}

	vinePort := remote.Port
	if vinePort == 0 {
		vinePort = 7633
	}

	tunnel := fmt.Sprintf("%d:127.0.0.1:%d", localPort, vinePort)

	target := remote.Host
	if remote.SSHUser != "" {
		target = remote.SSHUser + "@" + remote.Host
	}

	args := []string{
		"-N",
		"-o", "ExitOnForwardFailure=yes",
		"-o", "StrictHostKeyChecking=accept-new",
		"-o", "ServerAliveInterval=30",
		"-o", "ServerAliveCountMax=3",
		"-L", tunnel,
	}

	sshPort := remote.SSHPortOrDefault()
	if sshPort != 22 {
		args = append(args, "-p", strconv.Itoa(sshPort))
	}
	args = append(args, target)

	cmd := exec.Command("ssh", args...)
	// Detach from controlling terminal.
	cmd.SysProcAttr = &syscall.SysProcAttr{Setsid: true}
	cmd.Stdin = nil
	cmd.Stdout = nil
	cmd.Stderr = nil

	if err := cmd.Start(); err != nil {
		return nil, fmt.Errorf("starting SSH tunnel to %s: %w", remote.Host, err)
	}

	// Wait for the tunnel to be ready.
	localAddr := fmt.Sprintf("127.0.0.1:%d", localPort)
	if err := waitForPort(localAddr, 10*time.Second); err != nil {
		cmd.Process.Kill()
		cmd.Wait()
		return nil, fmt.Errorf("SSH tunnel to %s failed to start (is the host reachable and vine server running?)", remote.Host)
	}

	// Capture PID before releasing the process handle.
	pid := cmd.Process.Pid

	// Detach so the tunnel outlives this process.
	cmd.Process.Release()

	info := &TunnelInfo{
		Name:      remote.Name,
		PID:       pid,
		LocalPort: localPort,
		Host:      remote.Host,
		SSHUser:   remote.SSHUser,
		SSHPort:   sshPort,
		VinePort:  vinePort,
	}

	if err := saveTunnel(info); err != nil {
		// Tunnel is running but we couldn't save state — kill it.
		if proc, err := os.FindProcess(info.PID); err == nil {
			proc.Kill()
		}
		return nil, fmt.Errorf("saving tunnel state: %w", err)
	}

	return info, nil
}

// Disconnect tears down a persistent SSH tunnel for the named remote.
func Disconnect(name string) error {
	info, err := LoadTunnel(name)
	if err != nil {
		return fmt.Errorf("reading tunnel state: %w", err)
	}
	if info == nil {
		return fmt.Errorf("no active tunnel for %q", name)
	}

	defer removeTunnelFile(name)

	proc, err := os.FindProcess(info.PID)
	if err != nil {
		return nil // process already gone
	}

	if err := proc.Signal(syscall.SIGTERM); err != nil {
		return nil // process already gone
	}

	// Wait briefly for graceful exit.
	for i := 0; i < 20; i++ {
		time.Sleep(50 * time.Millisecond)
		if proc.Signal(syscall.Signal(0)) != nil {
			return nil
		}
	}

	proc.Signal(syscall.SIGKILL)
	return nil
}

// freePort asks the OS for a free TCP port.
func freePort() (int, error) {
	l, err := net.Listen("tcp", "127.0.0.1:0")
	if err != nil {
		return 0, err
	}
	port := l.Addr().(*net.TCPAddr).Port
	l.Close()
	return port, nil
}

// waitForPort polls until a TCP connection succeeds or the timeout expires.
func waitForPort(addr string, timeout time.Duration) error {
	deadline := time.Now().Add(timeout)
	for time.Now().Before(deadline) {
		conn, err := net.DialTimeout("tcp", addr, 500*time.Millisecond)
		if err == nil {
			conn.Close()
			return nil
		}
		time.Sleep(100 * time.Millisecond)
	}
	return fmt.Errorf("timeout waiting for %s", addr)
}

// ListTunnels returns info about all active tunnels.
func ListTunnels() ([]*TunnelInfo, error) {
	dir, err := tunnelsDir()
	if err != nil {
		return nil, err
	}

	entries, err := os.ReadDir(dir)
	if err != nil {
		if os.IsNotExist(err) {
			return nil, nil
		}
		return nil, err
	}

	var tunnels []*TunnelInfo
	for _, e := range entries {
		if filepath.Ext(e.Name()) != ".json" {
			continue
		}
		name := e.Name()[:len(e.Name())-5]
		info, err := LoadTunnel(name)
		if err != nil || info == nil {
			continue
		}
		tunnels = append(tunnels, info)
	}
	return tunnels, nil
}
