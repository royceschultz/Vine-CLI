---
name: vine-implement
description: Pick up a single vine task, implement it, commit, and report back with status and next steps.
---

# Vine Implement (Single Task)

Implement a single vine task, commit the work, and report back to the user with status and next steps.

## Workflow

1. **Pick up the task.** The user will specify a task, or you can look at what's ready with `vine ready`. Use `vine show <id>` to understand the full scope. Claim it with `vine pick <id>`.
2. **Ask questions first.** If the task description is ambiguous or design decisions are unclear, ask the user before writing code. Don't guess at requirements — it's cheaper to ask than to rework.
3. **Implement the task.** Write the code. Keep your focus on completing this task rather than getting sidetracked by adjacent improvements.
4. **Commit at checkpoints.** Make commits at meaningful points — not after every line, but not as one giant commit either. See the committing section below.
5. **Surface testable work.** When the implementation reaches a point where something is ready for the user to test, explicitly say so. See the testing section below.
6. **Create follow-up tasks if needed.** If the task turns out to be larger than expected or you discover necessary future work, create follow-up tasks in vine. But keep this balanced — your primary job is implementing, not project management. See the follow-up section below.
7. **Report back.** When done, close the task and give the user a clear summary: what you did, what's ready to test, and what's next.

## Committing

Make commits at important checkpoints, sized appropriately — not too small, not too large.

**When parallel work causes conflicts:** You may be working on a branch where others (or other agents) are making changes in parallel. This can cause confusion at commit time — e.g., a file you changed was already committed by someone else. When this happens:

- Don't panic. Quickly verify your work is still intact (a brief check, not an exhaustive audit).
- Commit the remaining files you changed.
- Roll with the messy git history — it's fine.
- **Do not git stash** to work around conflicts unless you have explicit permission from the user. Stashing in a parallel-work environment causes more confusion than it solves.

## Testing

Explicitly tell the user when something is ready for them to test. Not all tasks need manual testing:

- **Backend/infra tasks** can often be validated programmatically — run the tests and report results.
- **Frontend/UX tasks** often require the user to go through the workflow and validate behavior. Tell them exactly what to try: "You can now test this by going to X and doing Y."

Don't wait until the very end to mention testing. If a meaningful piece is testable mid-implementation, say so.

## Follow-up Tasks

If you discover the task is bigger than expected, or find related work that needs doing later, create follow-up tasks in vine. This is a **last resort** for when you know more work is necessary but it shouldn't block completing the current task. Don't create follow-ups for things you can just do now. The bar is: "we've discovered future work that doesn't belong in this task."

**Place follow-ups correctly in the task tree.** Check if the current task has a parent (`vine show <current-task-id>`). Follow-ups should be:

- **Siblings of the current task** (same parent) when the follow-up is peer-level work within the same feature/epic.
- **Children of the current task** when the follow-up is a piece of the current task that you're deferring.

```bash
# Follow-up as sibling (most common — use the current task's parent)
vine create "Follow-up title" -t task -d "Description" --parent <current-tasks-parent-id>

# Follow-up as child of the current task
vine create "Follow-up title" -t task -d "Description" --parent <current-task-id>
```

**Set dependencies** when the follow-up has ordering constraints:

```bash
# Follow-up depends on the current task finishing first
vine dep add <follow-up-id> <current-task-id>

# Follow-up must be done before another existing task
vine dep add <other-task-id> <follow-up-id>
```

Not every follow-up needs a dependency — only add them when there's a real ordering constraint.

## Key Commands

```bash
vine ready                               # tasks ready to work on
vine show <id>                           # full task details
vine pick <id>                           # claim a task (sets it to in_progress)
vine close <id>                          # mark task as done
vine create "Title" -t task -d "..." --parent <parent-id>   # create follow-up
vine children <parent-id>               # see sibling tasks for context
```

## Tips

- Use `vine show <id>` on the parent epic/feature to understand the broader context of the task you're implementing.
- Use `vine children <parent-id>` to see where your task fits among siblings.
- Ask the user questions when uncertain. A 30-second conversation saves 30 minutes of rework.
