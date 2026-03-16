---
name: vine-backlog-refine
description: Review and refine the vine backlog — group related tasks, add dependencies, and discuss priorities with the user.
---

# Vine Backlog Refinement

Review and clean up the current vine backlog using the `vine` CLI. The goal is to organize, group, prioritize, and improve task descriptions so the backlog stays useful and actionable.

## Workflow

1. **Survey the backlog.** Start by listing all open issues and epics to get the lay of the land.
2. **Group related tasks.** Look for tasks that belong together under a shared parent (epic, feature, or task). Create parent issues where needed and reparent orphaned or loosely related tasks.
3. **Add dependencies.** Where tasks have a natural ordering — one must be done before another — add dependency links.
4. **Discuss priorities with the user.** After reviewing the backlog, share your observations about what seems most important, what could be deferred, and what ordering makes sense. Ask the user for their input — don't just silently reorder things.
5. **Improve task descriptions.** If tasks are vague, underspecified, or too large, suggest breaking them down or expanding their descriptions. Discuss with the user before making changes.
6. **Identify stale or irrelevant tasks.** Flag tasks that may no longer be relevant and ask the user whether to close or remove them.

### What to discuss with the user

After surveying the backlog, have a brief conversation covering:

- **Priority observations**: "These epics/features seem like the most important next steps: X, Y, Z. Does that match your thinking?"
- **Suggested ordering**: "Even without hard dependencies, it might make sense to tackle A before B because..."
- **Stale items**: "These tasks haven't been touched in a while and might be outdated: ..."
- **Underspecified work**: "These tasks could use more detail or could be broken down further: ..."
- **Grouping proposals**: "These tasks seem related and could be grouped under a new epic/feature: ..."

This is a collaborative refinement — the agent should present findings and proposals, then act on the user's decisions.

### Organizing with parents and subtasks

- **Epics** group large bodies of work (e.g., "Authentication system").
- **Nested epics** break a large epic into sub-areas (e.g., "OAuth flow" and "Session management" as child epics under "Authentication system").
- **Any task type can have subtasks.** A feature like "Login page" can have child tasks for "form validation", "error states", etc.
- Think hierarchically: epic → sub-epic or feature → tasks → subtasks. Use the depth that matches the complexity of the work.
- Don't over-nest. If a grouping only has one or two children, it may not need its own parent.

## Key Commands

### Survey the backlog

```bash
vine list --root                     # top-level issues only
vine list                            # all open issues
vine list -t epic                    # list all epics
vine list -s ready                   # tasks ready to work on
vine list -s blocked                 # tasks waiting on dependencies
vine list --grep "search term"       # filter by task name
vine status                          # project summary with counts
```

### Inspect an issue

```bash
vine show <id> [id...]               # full details (accepts multiple IDs)
vine children <id>                   # list children/subtasks of an issue
```

### Group tasks under a parent

Create a new parent if needed:

```bash
vine create "Parent title" -t epic -d "Description"
```

Then reparent existing tasks:

```bash
vine subtask add <parent-id> <child-id>
```

### Add dependencies

```bash
vine dep add <blocked-id> <blocker-id>
```

This means `blocked-id` cannot start until `blocker-id` is completed. The first argument is the task that will be blocked.

### Update task details

```bash
vine update <id> -d "Improved description"
vine update <id> --name "Better title"
```

### Close irrelevant tasks

```bash
vine close <id>
```

### Break down a large task

```bash
vine create "Subtask title" -t task -d "What to do" --parent <parent-id>
```

## Tips

- **Don't spend too long reading code.** This is a high-level backlog refinement. The tasks should mostly speak for themselves. A quick search is fine if you need context, but keep the focus on organizing and discussing.
- **Discuss before acting.** Present your grouping, priority, and cleanup proposals to the user before making bulk changes. This is a collaborative process.
- Use `vine show <id>` to inspect any issue whose title isn't self-explanatory.
- Use `vine create "title" --json` to get JSON output for scripting.
- Use `vine children <id>` to see all subtasks of any parent, not just epics.
