# vine

Task tracking for AI agents. Pick off tasks like grapes on a vine.

Vine is a CLI-based project management tool backed by SQLite. It's designed to give AI coding agents (Claude Code, Cursor, Copilot) structured task context — what to work on, what's blocked, and how tasks relate — without leaving the terminal.

## Install

Ensure `~/go/bin` is in your PATH:

```sh
export PATH="$HOME/go/bin:$PATH"  # add to ~/.zshrc or ~/.bashrc
```

From the repo root:

```sh
go install .
```

There is no hot reloading. Go compiles to a binary, so you need to re-run `go install .` after every change.

## Quick start

```sh
vine init                              # set up a new project
vine create "Fix login bug" -t bug     # create a task
vine create "Add auth" -d "OAuth2 flow" -t feature
vine ready                             # see what's ready to work on
vine pick <id>                         # claim a task (open → in_progress)
vine close <id>                        # mark done
vine status                            # project summary
```

## Concepts

**Tasks** have a name, optional description and details, a type (`task`, `bug`, `feature`, `epic`), and a status (`open`, `in_progress`, `done`, `cancelled`). Each task gets a short random ID (e.g. `k7x2a`) displayed with a project prefix like `myproject-k7x2a`. In output, `open` tasks are displayed as either `ready` or `blocked` depending on whether their dependencies are satisfied.

**Subtasks** form a parent-child hierarchy. Any task can be a parent. Use this to break epics into features, features into tasks.

**Dependencies** define ordering. A task can depend on other tasks — it won't appear in `vine ready` until its dependencies are done or cancelled.

**Tags** are free-form labels attached to tasks for filtering.

**Comments** are timestamped notes on tasks. Closing/cancelling/reopening with `--reason` auto-creates typed comments.

## Commands

### Task lifecycle

| Command | Description |
|---------|-------------|
| `vine create <name>` | Create a task (`-d` description, `-t` type, `-p` parent, `--tag`) |
| `vine pick <id>` | Claim a task (sets status to `in_progress`) |
| `vine close <id> [-r reason]` | Mark done (accepts multiple IDs) |
| `vine cancel <id> [-r reason]` | Cancel a task (accepts multiple IDs) |
| `vine reopen <id> [-r reason]` | Reopen a closed/cancelled task |
| `vine update <id>` | Update fields (`--name`, `-d`, `--details`, `-t`, `--add-tag`, `--rm-tag`) |

### Viewing tasks

| Command | Description |
|---------|-------------|
| `vine show <id>` | Full task detail — deps, subtasks, tags, parent chain, timestamps |
| `vine show <id> --short` | Minimal one-liner |
| `vine show <id> --detailed` | Includes metadata and comments |
| `vine list` | All active tasks (hides done/cancelled by default) |
| `vine list -s <status>` | Filter by status (`open`, `in_progress`, `done`, `cancelled`; open displays as ready/blocked) |
| `vine list -t <type>` | Filter by type |
| `vine list --tag <name>` | Filter by tag |
| `vine list --all` | Include done and cancelled |
| `vine search <query>` | Search names, descriptions, and details |

### Workflow views

| Command | Description |
|---------|-------------|
| `vine ready` | Tasks that are open and have no unsatisfied dependencies |
| `vine blocked` | Tasks that are waiting on other tasks |
| `vine status` | Count of tasks by status |
| `vine status --detailed` | Status counts with type breakdown |

### Hierarchy

| Command | Description |
|---------|-------------|
| `vine subtask add <parent> <child>` | Make a task a subtask of another |
| `vine subtask remove <child>` | Detach from parent (make root) |
| `vine subtask list <parent>` | List subtasks |
| `vine children <id>` | List children of a task |
| `vine parent <id>` | Show ancestor chain |

### Dependencies

| Command | Description |
|---------|-------------|
| `vine dep add <task> <depends-on>` | Task is blocked until dependency is done |
| `vine dep remove <task> <depends-on>` | Remove a dependency |
| `vine dep list <task>` | What this task depends on |
| `vine dep dependents <task>` | What tasks are waiting on this one |

### Tags and comments

| Command | Description |
|---------|-------------|
| `vine tags list` | All tags with task counts |
| `vine tags prune` | Remove orphan tags |
| `vine comment add <id> <message>` | Add a comment |
| `vine comment list <id>` | List comments |
| `vine comment delete <comment-id>` | Delete a comment |

### Project management

| Command | Description |
|---------|-------------|
| `vine init` | Initialize a new project (interactive storage and git-tracking selection) |
| `vine init --storage=local --git-tracked=false` | Non-interactive init for scripting/CI |
| `vine config` | Interactively reconfigure the current project |
| `vine config show` | Display current project configuration |
| `vine config set <key> <value>` | Update a configuration value |
| `vine db list` | List local and global databases |
| `vine db rename <name>` | Rename the current project's database |
| `vine doctor` | Diagnose config and integration issues |
| `vine doctor --fix` | Auto-fix issues where possible |
| `vine migrate` | Move or merge databases between local and global storage |
| `vine prune` | Clean up temporary files (logs, PID files) |
| `vine symlink create` | Create symlinks from `.vine/` to the global database |

## Flags

Most commands support `--json` for machine-readable output. Listing commands support `--root` to show only top-level tasks, and `-n` to limit results.

Global flags available on all commands:

- `--project <name>` — query a global database by name, without needing a `.vine/config` in the current directory.
- `--remote <name>` — query a remote vine server instead of the local database. See [Remote Access](docs/remote-access.md).

## Storage modes

Vine supports two storage modes, chosen during `vine init`:

- **Local** — database lives at `.vine/vine.db` inside the project. Simple, self-contained.
- **Global** — database lives at `~/.vine/databases/<name>.db`. Shared across git worktrees. Symlinks are created in `.vine/` so file watchers still detect changes.

`vine init` also asks whether to track `.vine/` in git. By default it adds a `.gitignore` so tasks stay local; opt in to share the backlog via version control.

You can switch storage modes later with `vine migrate`.

## AI agent integration

Vine is built to be used by AI agents as their task management backbone.

### Claude Code

```sh
vine init claude            # system prompt + SessionStart/PreCompact hooks
vine init claude --no-hooks # system prompt only, skip hooks
```

This writes into `.claude/settings.local.json`:
- A **system prompt** telling Claude to run `vine onboard` at session start.
- **Hooks** (`SessionStart` and `PreCompact`) that automatically run `vine prime` to inject task context into the conversation.

### Cursor / Copilot

```sh
vine init cursor
vine init copilot
```

### Key commands for agents

- `vine prime` — outputs a structured context block with project status, ready tasks, in-progress tasks, and a command reference. Designed to be consumed at the start of a session.
- `vine onboard` — prints a quick-start guide for agents new to the project.
- `vine ready --json` — machine-readable list of actionable tasks.
- `vine show <id> --json` — full task detail with relations as structured data.

## Publishing

Before publishing for `go install` to work remotely:

1. Choose a hosting URL (e.g., `github.com/yourorg/vine`)
2. Update the module path in `go.mod`
3. Find and replace `"vine/` with `"github.com/yourorg/vine/` in all `.go` imports
4. Run `go mod tidy && go build ./...` to verify
5. Update the install command:

```sh
go install github.com/yourorg/vine@latest
```
