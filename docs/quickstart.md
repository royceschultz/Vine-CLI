# Quickstart

Vine is a task tracker designed for AI agents. It stores tasks in a local SQLite database and gives agents structured context about what to work on, what's blocked, and what's done. This guide walks through setting up a project and using vine day-to-day.

## Install

```
go install vine@latest
```

## Initialize a project

Run `vine init` in your project root. It asks two questions:

1. **Storage mode** — where to put the SQLite database.
   - **Local** (default): `.vine/vine.db` inside your project directory. Simple, self-contained.
   - **Global**: `~/.vine/databases/<name>.db`. Useful when you work across multiple git worktrees and want them to share the same task database.

2. **Git tracking** — whether to commit `.vine/` to version control.
   - **No** (default): adds a `.gitignore` inside `.vine/` so it stays local.
   - **Yes**: tasks travel with the repo. Good for shared projects where everyone should see the backlog.

For scripting or CI, skip the interactive prompts:

```
vine init --storage=local --git-tracked=false
```

## Set up Claude Code integration

```
vine init claude
```

This writes two things into `.claude/settings.local.json` (project-local, won't affect other repos):

- A **system prompt** telling Claude to run `vine onboard` at the start of each session.
- **Session hooks** (`SessionStart` and `PreCompact`) that automatically run `vine prime` to inject task context into Claude's conversation.

The hooks are the important part — they mean Claude always knows the current project status without being asked. If you need the system prompt without hooks for some reason, pass `--no-hooks`.

## Create tasks

```
vine create "Add user authentication" -d "OAuth2 flow with Google and GitHub" -t feature
```

- `-d` sets a short description (shown in listings).
- `--details` sets longer technical notes (shown in `vine show`).
- `-t` sets the type: `task` (default), `feature`, `bug`, or `epic`.
- `-p` nests it under a parent task.
- `--tag` adds tags (repeatable or comma-separated).

Tasks start with status `open`.

## Organize with hierarchy and dependencies

Break large work into subtasks:

```
vine create "Set up OAuth provider config" -p vine-abc12
vine create "Implement callback handler" -p vine-abc12
vine create "Add session middleware" -p vine-abc12
```

A parent task can't be closed until all its subtasks are done or cancelled.

Add dependencies to control ordering:

```
vine dep add vine-xyz99 vine-abc12
```

This means `vine-xyz99` is **blocked** until `vine-abc12` is done. Blocked tasks won't appear in `vine ready`.

## The task lifecycle

```
open ──> in_progress ──> done
  │                        │
  └──> cancelled     <─────┘ (via reopen)
```

| Command | What it does |
|---|---|
| `vine pick <id>` | Move from `open` to `in_progress`. Records the current git branch and directory. |
| `vine close <id>` | Move to `done`. Fails if there are incomplete subtasks. Use `-r "reason"` to record why. |
| `vine cancel <id>` | Move to `cancelled`. Also accepts `-r`. |
| `vine reopen <id>` | Move back to `open` from `done` or `cancelled`. |

## Find what to work on

`vine ready` is the primary command. It shows tasks that are:
- Status `open` (not yet picked up)
- Not blocked by any incomplete dependency

```
$ vine ready
5 ready:

  vine-abc12  Add user authentication [feature]
              OAuth2 flow with Google and GitHub
  vine-def34  Fix login redirect loop [bug]
              Users get stuck after password reset
  ...
```

Other views:

| Command | Shows |
|---|---|
| `vine list` | All open/in-progress tasks (add `--all` for done/cancelled) |
| `vine list -s in_progress` | Just in-progress work |
| `vine list -t bug` | Just bugs |
| `vine list --tag backend` | Just tasks tagged "backend" |
| `vine blocked` | Tasks waiting on dependencies |
| `vine search <keyword>` | Full-text search across names, descriptions, and details |
| `vine status` | Summary counts by status |

## Inspect a task

```
$ vine show vine-abc12
vine-abc12  in_progress  Add user authentication
  type:  feature

  OAuth2 flow with Google and GitHub

  subtasks:
    vine-ghi56  open  Set up OAuth provider config
    vine-jkl78  done  Implement callback handler
    vine-mno90  open  Add session middleware

  blocks:
    vine-xyz99  open  Add user profile page

  created: 2026-03-01 10:00:00    updated: 2026-03-08 14:30:00
```

Add `--detailed` to include metadata (which branch/directory it was created from) and comments.

## Tags and comments

```
vine update vine-abc12 --add-tag auth --add-tag backend
vine comment add vine-abc12 "Decided to use PKCE flow instead of implicit"
vine comment list vine-abc12
```

Remove tags with `--rm-tag`. Tags with no tasks can be cleaned up with `vine tags prune`.

## Update tasks

```
vine update vine-abc12 --name "Add OAuth2 authentication" -d "New description" -t feature
```

All fields are optional — only the ones you pass get changed.

## Storage concepts

### Local vs global storage

**Local** stores everything in `.vine/vine.db`. This is the simplest setup. Each worktree has its own database.

**Global** stores the database in `~/.vine/databases/<name>.db` and creates symlinks from `.vine/` pointing to it. Multiple worktrees can point to the same database by using the same name during `vine init`. Run `vine db list` to see all global databases.

You can switch modes later with `vine config set storage <mode>`, which handles migrating the database.

### The `--project` flag

Access any global database without a `.vine/` directory:

```
vine --project myapp list
vine --project myapp ready
```

This is useful for checking on a project from anywhere.

## Machine-readable output

Every command supports `--json` for structured output, which makes vine easy to integrate with scripts or other tools:

```
vine --json ready
vine --json show vine-abc12
vine --json list -s in_progress
```

## Diagnostics

```
vine doctor
```

Checks your vine setup: project config, database health, Claude Code integration (system prompt and hooks), symlinks, and disk usage. Run `vine doctor --fix` to auto-repair symlinks and other fixable issues.

```
vine prune
```

Cleans up stale PID files and rotated server logs in `~/.vine/`.

## How AI agents use vine

When Claude Code starts a session in a vine-enabled project, the hooks automatically run `vine prime`, which outputs:

- Project name and task counts by status
- Up to 10 ready tasks with descriptions
- Any in-progress tasks
- A command reference

This gives the agent immediate context. From there, a typical agent workflow is:

1. `vine ready` — see what's available
2. `vine pick <id>` — claim a task
3. `vine show <id>` — read the details
4. *(do the work)*
5. `vine close <id> -r "implemented in commit abc123"` — mark done
6. `vine ready` — pick the next one

The `vine onboard` command prints a shorter quick-reference that agents can use if they need a refresher on available commands.
