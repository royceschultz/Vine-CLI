---
name: vine-implement-auto
description: Autonomously work through multiple vine tasks in sequence — implement, commit, and move on to the next task.
---

# Vine Implement (Autonomous Multi-Task)

Work through multiple vine tasks autonomously. Implement each task, commit, and move on to the next one. Check in with the user at key decision points rather than after every task.

## Workflow

1. **Plan the session.** Start by reviewing what's ready: `vine ready`. Propose an order to the user and get their sign-off before diving in. If the user specified particular tasks, start with those.
2. **For each task:**
   a. **Read the task.** Use `vine show <id>` to understand scope. Claim it with `vine pick <id>`.
   b. **Ask questions if needed.** If design decisions are unclear, ask before writing code. Don't guess at requirements. This is the one thing that should interrupt your flow — better to ask than to build the wrong thing.
   c. **Implement it.** Write the code, staying focused on the task at hand.
   d. **Commit.** Make appropriately-sized commits at meaningful checkpoints. See the committing section below.
   e. **Flag testable work.** When something is ready for the user to test, explicitly say so — don't bury it. See the testing section below.
   f. **Create follow-ups if needed.** If the task is larger than expected or reveals future work, create follow-up tasks. But stay focused on implementing — project management is a last resort. See the follow-up section below.
   g. **Close the task.** `vine close <id>` and move on. If this completes the last open sub-task under a parent, review the parent and consider closing it too.
3. **Check in periodically.** After completing a few tasks, give the user a brief status update: what's done, what's testable, and what's next.

## Committing

Make commits at important checkpoints, sized appropriately — not too small, not too large. When working through multiple tasks, commit before moving on to the next task at minimum.

**When parallel work causes conflicts:** You may be working on a branch where others (or other agents) are making changes in parallel. This can cause confusion at commit time — e.g., a file you changed was already committed by someone else. When this happens:

- Don't panic. Quickly verify your work is still intact (a brief check, not an exhaustive audit).
- Commit the remaining files you changed.
- Roll with the messy git history — it's fine.
- **Do not git stash** to work around conflicts unless you have explicit permission from the user. Stashing in a parallel-work environment causes more confusion than it solves.

## Testing

Explicitly tell the user when something is ready for them to test. Not all tasks need manual testing:

- **Backend/infra tasks** can often be validated programmatically — run the tests and report results.
- **Frontend/UX tasks** often require the user to go through the workflow and validate behavior. Tell them exactly what to try: "You can now test this by going to X and doing Y."

Don't batch all testing notes to the end. If a task produces something testable, mention it when you close that task — even if you're continuing on to the next one.

## Follow-up Tasks

If you discover a task is bigger than expected, or find related work that needs doing later, create follow-up tasks in vine. This is a **last resort** for when you know more work is necessary but it shouldn't block the current task. Don't create follow-ups for things you can just do now. The bar is: "we've discovered future work that doesn't belong in this task."

Balance individual productivity with awareness. Creating a task takes 10 seconds. Forgetting about necessary work costs much more. But don't let task management distract from the actual implementation.

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
vine children <parent-id>               # see subtasks for context
vine list -t epic                        # check epic list
vine status                              # check overall project progress
```

## Tips

- Use `vine show <id>` on the parent epic/feature to understand the broader context.
- Use `vine children <parent-id>` to see where tasks fit among siblings and pick the next one.
- When in doubt about a design decision, ask. Autonomy means working independently, not guessing at requirements.
- If you hit a blocker on one task, consider moving to another unblocked task and flagging the blocker for the user.
