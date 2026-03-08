package config

import (
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
)

// Remote represents a configured remote vine server.
type Remote struct {
	Name      string `json:"name"`
	Host      string `json:"host"`
	Port      int    `json:"port"`
	Token     string `json:"token,omitempty"`
	TLS       bool   `json:"tls,omitempty"`
	Transport string `json:"transport,omitempty"` // "ssh" (default) or "http"
	SSHUser   string `json:"ssh_user,omitempty"`
	SSHPort   int    `json:"ssh_port,omitempty"` // default 22
}

// IsSSH returns true if this remote uses SSH tunnel transport (the default).
func (r *Remote) IsSSH() bool {
	return r.Transport == "" || r.Transport == "ssh"
}

// SSHPortOrDefault returns the configured SSH port, or 22 if not set.
func (r *Remote) SSHPortOrDefault() int {
	if r.SSHPort > 0 {
		return r.SSHPort
	}
	return 22
}

// Scheme returns "https" if TLS is enabled, otherwise "http".
func (r *Remote) Scheme() string {
	if r.TLS {
		return "https"
	}
	return "http"
}

// URL returns the base URL for this remote (e.g. "http://host:port").
func (r *Remote) URL() string {
	return fmt.Sprintf("%s://%s:%d", r.Scheme(), r.Host, r.Port)
}

// RemotesConfig is the top-level structure of ~/.vine/remotes.json.
type RemotesConfig struct {
	Remotes []Remote `json:"remotes"`
}

func remotesPath() (string, error) {
	home, err := os.UserHomeDir()
	if err != nil {
		return "", fmt.Errorf("resolving home directory: %w", err)
	}
	return filepath.Join(home, ".vine", "remotes.json"), nil
}

// LoadRemotes reads ~/.vine/remotes.json. Returns an empty config if the file doesn't exist.
func LoadRemotes() (*RemotesConfig, error) {
	path, err := remotesPath()
	if err != nil {
		return nil, err
	}

	data, err := os.ReadFile(path)
	if err != nil {
		if os.IsNotExist(err) {
			return &RemotesConfig{}, nil
		}
		return nil, fmt.Errorf("reading remotes config: %w", err)
	}

	var cfg RemotesConfig
	if err := json.Unmarshal(data, &cfg); err != nil {
		return nil, fmt.Errorf("parsing remotes config: %w", err)
	}
	return &cfg, nil
}

// SaveRemotes writes the remotes config to ~/.vine/remotes.json.
func SaveRemotes(cfg *RemotesConfig) error {
	path, err := remotesPath()
	if err != nil {
		return err
	}

	if err := os.MkdirAll(filepath.Dir(path), 0o755); err != nil {
		return fmt.Errorf("creating ~/.vine directory: %w", err)
	}

	data, err := json.MarshalIndent(cfg, "", "  ")
	if err != nil {
		return fmt.Errorf("marshaling remotes config: %w", err)
	}
	return os.WriteFile(path, append(data, '\n'), 0o644)
}

// GetRemote returns the remote with the given name, or nil if not found.
func (cfg *RemotesConfig) GetRemote(name string) *Remote {
	for i := range cfg.Remotes {
		if cfg.Remotes[i].Name == name {
			return &cfg.Remotes[i]
		}
	}
	return nil
}

// AddRemote adds a remote to the config. Returns an error if the name already exists.
func (cfg *RemotesConfig) AddRemote(r Remote) error {
	if cfg.GetRemote(r.Name) != nil {
		return fmt.Errorf("remote %q already exists", r.Name)
	}
	cfg.Remotes = append(cfg.Remotes, r)
	return nil
}

// RemoveRemote removes a remote by name. Returns an error if not found.
func (cfg *RemotesConfig) RemoveRemote(name string) error {
	for i, r := range cfg.Remotes {
		if r.Name == name {
			cfg.Remotes = append(cfg.Remotes[:i], cfg.Remotes[i+1:]...)
			return nil
		}
	}
	return fmt.Errorf("remote %q not found", name)
}
