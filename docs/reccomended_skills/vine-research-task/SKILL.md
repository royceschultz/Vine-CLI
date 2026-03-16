---
name: vine-research-task
description: Research a specific vine task — explore the codebase, clarify scope, break it down, and refine the description so it's ready to implement.
---

# Vine Research Task

Deep-dive into a specific vine task to understand what's involved, refine its description, break it down if needed, and make it implementation-ready.

## Workflow

1. **Understand the task.** Use `vine show <id>` to read the task and its context. Check its parent (`vine show <parent-id>`) and siblings (`vine children <parent-id>`) to understand where it fits.
2. **Research the codebase.** Explore the relevant code to understand the current state. What exists today? What would need to change? Where are the touch points? Keep this focused — you're scoping, not implementing.
3. **Identify unknowns and decisions.** Flag anything that's ambiguous, has multiple possible approaches, or needs a design decision. Present these to the user.
4. **Discuss with the user.** Share what you've learned and propose a plan. Cover:
   - **Scope**: "Here's what I think this task involves: ..."
   - **Approach options**: "There are a few ways to do this: A (simpler, but...) vs B (more flexible, but...)"
   - **Open questions**: "Before this can be implemented, we need to decide: ..."
   - **Estimated complexity**: "This looks straightforward / medium / larger than expected because..."
5. **Refine the task.** Based on the discussion, update the task description to capture the agreed-upon scope and approach. Add implementation notes that will be useful to whoever picks it up.
6. **Break down if needed.** If the task is too large for a single implementation pass, break it into subtasks with clear boundaries. Set dependencies where ordering matters.
7. **Flag blockers.** If the task depends on other work that isn't done yet, add dependency links and let the user know.

## Key Commands

### Inspect the task and its context

```bash
vine show <id>                       # full task details
vine show <parent-id>                # understand the broader context
vine children <parent-id>            # see sibling tasks
vine dep list <id>                   # check existing dependencies
```

### Update the task after research

```bash
vine update <id> -d "Refined description with implementation notes"
vine update <id> --name "More precise title"
```

### Break down into subtasks

```bash
vine create "Subtask title" -t task -d "Description" --parent <id>
vine dep add <later-subtask-id> <earlier-subtask-id>
```

### Add dependencies on other work

```bash
vine dep add <this-task-id> <blocker-task-id>
```

## Tips

- **This is research, not implementation.** Read code to understand scope, but don't start writing production code. The output is a well-scoped, well-described task (or set of subtasks), not a commit.
- **Focus on the specific task.** Don't get pulled into refining the whole backlog. Stay focused on making this one task implementation-ready.
- **Capture what you learn.** Put your findings into the task description so the context isn't lost. Implementation notes like "the relevant code is in X, the main change would be Y" are valuable.
- **Ask the user questions.** This is a collaborative research process. Surface decisions and tradeoffs rather than making assumptions silently.
- **Don't over-break-down.** Only split a task if it's genuinely too large for one implementation pass. Three focused subtasks are better than ten trivial ones.
