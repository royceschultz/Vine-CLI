# Remote Access

Vine supports querying tasks on remote machines. By default, connections are secured through SSH tunneling — the vine server binds to localhost on the remote machine and the client automatically creates an SSH tunnel to reach it.

## Architecture

```
┌─────────────────┐     SSH tunnel + HTTP     ┌─────────────────┐
│  Local Machine   │  ◄──────────────────►   │  Remote Server   │
│                  │                          │                  │
│  vine CLI        │  ssh -L port:lo:port     │  vine remote     │
│  vine manager    │  ──────────────────►     │  serve           │
│  (Electron app)  │  HTTP over tunnel        │  127.0.0.1:7633  │
│                  │                          │  ~/.vine/        │
│                  │                          │  databases/*.db  │
└─────────────────┘                          └─────────────────┘
```

## Quick Start

### On the remote machine (server)

```bash
# Start the server (binds to localhost by default, daemonizes)
vine remote serve
```

That's it. The server listens on `127.0.0.1:7633` — SSH handles the rest.

### On your local machine (client)

```bash
# Add the remote (SSH transport is the default)
vine remote add desktop 192.168.1.100

# Test connectivity (establishes SSH tunnel automatically)
vine remote ping desktop

# List available projects
vine remote projects desktop

# Query tasks
vine list --remote desktop --project my-project
vine status --remote desktop --project my-project
```

## Connection Methods

### SSH tunnel (default)

The recommended and default method. The client automatically creates an SSH tunnel using your existing SSH keys/agent, then sends HTTP requests through the tunnel. The vine server never needs to listen on a public interface.

```bash
# Basic — uses your default SSH user and key
vine remote add desktop 192.168.1.100

# With explicit SSH user
vine remote add desktop 192.168.1.100 --ssh-user royce

# Non-standard SSH or vine port
vine remote add desktop 192.168.1.100 --ssh-port 2222 --port 8080
```

**Requirements:**
- SSH access to the remote host (key-based auth recommended)
- The vine server running on the remote host (`vine remote serve`)

**How it works:**
1. On first use (or via `vine remote connect`), the client picks a free local port
2. Runs `ssh -N -L <localport>:127.0.0.1:7633 <host>` as a background process
3. Saves tunnel state to `~/.vine/tunnels/<name>.json`
4. Subsequent commands reuse the existing tunnel — no per-command overhead
5. The tunnel persists until explicitly disconnected or the SSH connection drops

**Tunnel management:**

```bash
vine remote connect desktop       # explicitly open tunnel (optional — auto-connects on first query)
vine remote disconnect desktop    # close tunnel
vine remote disconnect-all        # close all tunnels
vine remote list                  # shows connection status (connected/disconnected)
```

If you take your laptop on the go, the tunnel will eventually die on its own (SSH keepalives detect the broken connection). The next query will auto-reconnect if the remote becomes reachable again. Stale tunnel state is cleaned up automatically.

### Direct HTTP (opt-in)

For environments where SSH isn't available or you want to expose vine directly (e.g., behind a reverse proxy). Less secure than SSH — use token auth and TLS.

```bash
# Direct HTTP with token auth
vine remote add cloud api.example.com --http --token s3cret

# Direct HTTPS
vine remote add cloud api.example.com --http --token s3cret --tls
```

When using direct HTTP, the server must bind to a network interface:

```bash
# On the server — bind to all interfaces with token auth
vine remote serve --bind 0.0.0.0 --token my-secret-token

# With TLS
vine remote serve --bind 0.0.0.0 --token my-secret-token \
  --tls-cert /path/to/cert.pem --tls-key /path/to/key.pem
```

## Server Setup

### Starting the server

```bash
# Default: localhost only, port 7633, daemonizes
vine remote serve

# Foreground mode (useful for debugging or service managers)
vine remote serve --foreground

# Custom port
vine remote serve --port 8080
```

### Daemon management

```bash
vine remote serve     # start (daemonizes)
vine remote status    # check if running
vine remote stop      # graceful shutdown
vine remote restart   # stop + start with saved config
vine remote logs      # view server log (last 50 lines)
vine remote logs -f   # follow log output
```

**PID file:** `~/.vine/server.pid`
**Log file:** `~/.vine/server.log` (rotates at 10MB, keeps 3 old files)
**Config file:** `~/.vine/server.json` (saved on start, used by restart)

### Process safeguards

- **Duplicate start:** Refuses to start if a server is already running, reports existing PID.
- **Stale PID file:** Detects dead processes and cleans up automatically.
- **PID reuse:** Verifies via `/api/health` that the PID belongs to an actual vine server.
- **Orphan detection:** Probes the target port to find servers running without a PID file.

## Client Configuration

### Managing remotes

```bash
vine remote add <name> <host>         # SSH tunnel (default)
vine remote add <name> <host> --http  # Direct HTTP
vine remote remove <name>
vine remote list                      # shows transport and connection status
vine remote ping <name>
vine remote projects <name>
vine remote connect <name>            # open persistent SSH tunnel
vine remote disconnect <name>         # close tunnel
vine remote disconnect-all            # close all tunnels
```

### Configuration file

Remote connections are stored in `~/.vine/remotes.json`:

```json
{
  "remotes": [
    {
      "name": "desktop",
      "host": "192.168.1.100",
      "port": 7633,
      "transport": "ssh",
      "ssh_user": "royce"
    },
    {
      "name": "cloud",
      "host": "api.example.com",
      "port": 7633,
      "transport": "http",
      "token": "my-secret-token",
      "tls": true
    }
  ]
}
```

## Querying Remote Data

Add `--remote <name> --project <project>` to any read command:

```bash
vine list --remote desktop --project myapp
vine list --remote desktop --project myapp --status open --type bug
vine show --remote desktop --project myapp abc12
vine show --remote desktop --project myapp abc12 --detailed
vine children --remote desktop --project myapp abc12
vine ready --remote desktop --project myapp
vine blocked --remote desktop --project myapp
vine status --remote desktop --project myapp --detailed
vine search --remote desktop --project myapp "login bug"
```

JSON output works the same way:

```bash
vine list --remote desktop --project myapp --json
vine status --remote desktop --project myapp --json
```

### Using --project without --remote

The `--project` flag also works locally to query a global database by name, without needing a `.vine/config` in the current directory:

```bash
vine status --project my-project
vine list --project my-project
```

## API Reference

The HTTP API is read-only and returns JSON. All endpoints are under `/api/`.

| Endpoint | Description |
|---|---|
| `GET /api/health` | Server health (always accessible, no auth) |
| `GET /api/projects` | List global database names |
| `GET /api/projects/{project}/tasks` | List tasks (query params: `status`, `type`, `tag`, `all`, `root`) |
| `GET /api/projects/{project}/tasks/{id}` | Get a single task |
| `GET /api/projects/{project}/tasks/{id}/children` | Child tasks |
| `GET /api/projects/{project}/tasks/{id}/ancestors` | Parent chain to root |
| `GET /api/projects/{project}/tasks/{id}/comments` | Task comments |
| `GET /api/projects/{project}/tasks/{id}/dependencies` | What the task depends on |
| `GET /api/projects/{project}/tasks/{id}/dependents` | What depends on the task |
| `GET /api/projects/{project}/tasks/{id}/tags` | Tags on the task |
| `GET /api/projects/{project}/ready` | Tasks ready to work on |
| `GET /api/projects/{project}/blocked` | Blocked tasks |
| `GET /api/projects/{project}/status` | Task summary (query param: `detailed`) |
| `GET /api/projects/{project}/search` | Search tasks (query param: `q`) |
| `GET /api/projects/{project}/tags` | All tags with counts |

### Authentication

When using direct HTTP with a token configured, include it in the `Authorization` header:

```bash
curl -H "Authorization: Bearer my-token" http://host:7633/api/projects
```

The `/api/health` endpoint is always accessible without authentication. With SSH transport, no token is needed — SSH handles authentication.

## Security Considerations

- **SSH by default:** Remotes use SSH tunneling unless `--http` is explicitly specified. SSH handles authentication and encryption using your existing keys.
- **Server binds to localhost:** The server binds to `127.0.0.1` by default. It cannot be reached from the network without explicit `--bind 0.0.0.0`.
- **Token auth for HTTP:** Only needed when using direct HTTP transport. Tokens are sent in plaintext over HTTP — use TLS if the network is untrusted.
- **Read-only:** The API only exposes read operations. No task creation, modification, or deletion is possible through the remote server.
- **Project name sanitization:** The server validates project names to prevent path traversal — only alphanumeric characters, dashes, underscores, and dots are allowed.
