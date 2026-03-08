# Remote Access

Vine supports querying tasks on remote machines over HTTP. This is useful when you run vine on a development server and want to view tasks from your local machine (e.g., in the vine manager Electron app).

## Architecture

```
┌─────────────────┐        HTTP/HTTPS        ┌─────────────────┐
│  Local Machine   │  ◄──────────────────►   │  Remote Server   │
│                  │                          │                  │
│  vine CLI        │   GET /api/projects/...  │  vine remote     │
│  vine manager    │                          │  serve           │
│  (Electron app)  │                          │  ~/.vine/        │
│                  │                          │  databases/*.db  │
└─────────────────┘                          └─────────────────┘
```

The remote server (`vine remote serve`) exposes a read-only HTTP API over vine's global databases. The client side uses `--remote` and `--project` flags to query it.

## Quick Start

### On the remote machine (server)

```bash
# Start the server (daemonizes by default)
vine remote serve

# Or with explicit options
vine remote serve --port 8080 --bind 0.0.0.0 --token my-secret-token
```

### On your local machine (client)

```bash
# Add the remote
vine remote add myserver 192.168.1.100 --port 8080 --token my-secret-token

# Test connectivity
vine remote ping myserver

# List available projects
vine remote projects myserver

# Query tasks
vine list --remote myserver --project my-project
vine status --remote myserver --project my-project
vine show --remote myserver --project my-project abc12
```

## Server Setup

### Basic (localhost only)

The simplest setup binds to localhost only. This is the default and is secure — no authentication is needed because the server is only reachable from the machine itself.

```bash
vine remote serve
# Listening on http://127.0.0.1:7633
```

To access this from another machine, use SSH port forwarding (see below).

### With token authentication

If you need to expose the server on a network interface, use `--bind` with `--token`:

```bash
vine remote serve --bind 0.0.0.0 --token my-secret-token
```

All API requests (except `/api/health`) will require a `Authorization: Bearer my-secret-token` header.

### With TLS

For encrypted connections:

```bash
vine remote serve --bind 0.0.0.0 --token my-secret-token \
  --tls-cert /path/to/cert.pem --tls-key /path/to/key.pem
```

### Daemon management

The server runs as a background daemon by default. Use `--foreground` to run in the foreground (useful for debugging or systemd/launchd integration).

```bash
# Start (daemon)
vine remote serve

# Stop
vine remote stop        # (coming soon — for now: kill $(cat ~/.vine/server.pid))

# Check status
vine remote status      # (coming soon — for now: vine remote ping <name>)

# View logs
cat ~/.vine/server.log
tail -f ~/.vine/server.log
```

**PID file:** `~/.vine/server.pid`
**Log file:** `~/.vine/server.log` (rotates at 10MB, keeps 3 old files)

### Process management

The server writes its PID to `~/.vine/server.pid` and includes safeguards for common issues:

- **Duplicate start:** If a server is already running, `vine remote serve` will refuse to start and tell you the existing PID.
- **Stale PID file:** If the server crashed and left behind a PID file, the next `serve` will detect that the process is gone and clean up automatically.
- **PID reuse:** The server exposes a `/api/health` endpoint that returns its PID and start time. When checking if a server is running, vine verifies the PID belongs to an actual vine server, not an unrelated process that reused the PID.
- **Orphan detection:** If the PID file is deleted while the server is running, `vine remote serve` will detect the orphan by probing the health endpoint on the target port and report the orphan's PID.

## SSH Port Forwarding

The recommended approach for secure remote access: run the server on localhost (default) and use SSH to forward the port.

```bash
# On your local machine, forward local port 7633 to the remote's localhost:7633
ssh -L 7633:localhost:7633 user@remote-host

# Then add a remote pointing to your local forwarded port
vine remote add myserver 127.0.0.1 --port 7633
```

This way:
- No token or TLS needed (SSH handles auth and encryption)
- The vine server never listens on a public interface
- You get SSH's authentication for free

For a persistent tunnel, use `ssh -fNL 7633:localhost:7633 user@remote-host` or configure it in `~/.ssh/config`:

```
Host myserver
    HostName remote-host
    User myuser
    LocalForward 7633 localhost:7633
```

## Client Configuration

Remote connections are stored in `~/.vine/remotes.json`:

```json
{
  "remotes": [
    {
      "name": "myserver",
      "host": "192.168.1.100",
      "port": 7633,
      "token": "my-secret-token",
      "tls": true
    }
  ]
}
```

### Managing remotes

```bash
vine remote add <name> <host> [--port 7633] [--token <token>] [--tls]
vine remote remove <name>
vine remote list
vine remote ping <name>
vine remote projects <name>
```

## Querying Remote Data

Add `--remote <name> --project <project>` to any read command:

```bash
vine list --remote myserver --project myapp
vine list --remote myserver --project myapp --status open --type bug
vine show --remote myserver --project myapp abc12
vine show --remote myserver --project myapp abc12 --detailed
vine children --remote myserver --project myapp abc12
vine ready --remote myserver --project myapp
vine blocked --remote myserver --project myapp
vine status --remote myserver --project myapp --detailed
vine search --remote myserver --project myapp "login bug"
```

JSON output works the same way:

```bash
vine list --remote myserver --project myapp --json
vine status --remote myserver --project myapp --json
```

### Using --project without --remote

The `--project` flag also works locally to query a global database by name, without needing a `.vine/config` in the current directory:

```bash
vine status --project my-project
vine list --project my-project
```

If you're in a directory with a `.vine/config`, a note will be printed to stderr indicating that the local config is being ignored.

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

### Health response

```json
{
  "service": "vine",
  "pid": 12345,
  "started_at": "2026-03-08T22:00:00Z"
}
```

### Authentication

When a token is configured, include it in the `Authorization` header:

```bash
curl -H "Authorization: Bearer my-token" http://host:7633/api/projects
```

The `/api/health` endpoint is always accessible without authentication.

## Security Considerations

- **Default is secure:** The server binds to `127.0.0.1` by default. It cannot be reached from the network without explicit `--bind 0.0.0.0`.
- **SSH forwarding preferred:** Use SSH tunnels for remote access when possible. This avoids exposing the server on a network interface entirely.
- **Token auth is optional:** Only needed when binding to non-localhost interfaces. Tokens are sent in plaintext over HTTP — use TLS if the network is untrusted.
- **Read-only:** The API only exposes read operations. No task creation, modification, or deletion is possible through the remote server.
- **Project name sanitization:** The server validates project names to prevent path traversal — only alphanumeric characters, dashes, underscores, and dots are allowed.
