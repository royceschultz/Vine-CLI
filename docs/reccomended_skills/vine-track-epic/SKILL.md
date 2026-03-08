---
name: vine-track-epic
description: Break down a high-level feature into incremental tasks and store them in vine, a project management tool.
---

# Vine Epic Tracking

Break down a high-level feature into incremental tasks using the `vine` CLI.

## Workflow

1. **Discuss design with the user.** Before creating tasks, talk through the feature with the user. Ask about design decisions, trade-offs, and open questions — e.g., "Should this use polling or WebSockets?", "Do we need to support the legacy API?", "What should happen when X fails?" Don't assume — surface ambiguities early so the breakdown reflects actual decisions, not guesses.
2. **Bucket work into epics and subtasks.** Group related tasks under epics. Any task type (feature, task, chore, bug, etc.) can also have subtasks — not just epics. Nest epics inside other epics when a feature has distinct sub-areas. Good organization of tasks and dependencies improves project progress and focus.
3. **Create tasks** as children of the appropriate parent for each incremental piece of work.
4. **Set dependencies** between tasks to define ordering.

### Organizing with nested epics and subtasks

- **Epics** group large bodies of work (e.g., "Authentication system").
- **Nested epics** break a large epic into sub-areas (e.g., "OAuth flow" and "Session management" as child epics under "Authentication system").
- **Any task type can have subtasks.** A feature like "Login page" can have child tasks for "form validation", "error states", etc. A chore like "Upgrade dependencies" can have child tasks per package.
- Think hierarchically: epic → sub-epic or feature → tasks → subtasks. Use the depth that matches the complexity of the work.

## Key Commands

### Create an epic

```bash
vine create "Epic title" -t epic -d "Description of the feature"
```

### Create tasks under any parent

```bash
vine create "Task title" -t task -d "What to do" --parent <parent-id>
```

The parent can be an epic, feature, task, chore, or any other issue type.

Other useful type values: `feature`, `bug`, `chore`.

### Add dependencies between tasks

```bash
vine dep add <blocked-id> <blocker-id>
```

This means `blocked-id` cannot start until `blocker-id` is completed. The first argument is the task that will be blocked.

### View epic status

```bash
vine list -t epic                      # list all epics
vine status                            # project summary with counts
vine show <epic-id>                    # details of a single epic
vine children <epic-id>                # subtasks of a single epic
```

### List open issues

```bash
vine list                              # all open issues
vine children <epic-id>                # children of a specific epic
```

### Close a completed task

```bash
vine close <issue-id>
```

## Tips

- **Bucket aggressively.** When breaking down work, group related tasks under a shared parent. If a set of tasks all serve one feature, create that feature first and nest them. This keeps `vine list` clean and makes progress visible.
- Use `vine create "title" --json` to get JSON output for scripting.
- Use `vine show <id>` to inspect any issue in detail.
- Use `vine children <id>` to see all subtasks of any parent, not just epics.
