package store

import (
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"vine/utils"
)

func TestCreateTask_Minimal(t *testing.T) {
	s := newTestStore(t)

	task, err := s.CreateTask(CreateTaskParams{Name: "my task"})
	require.NoError(t, err)

	assert.Len(t, task.ID, utils.IDLength)
	assert.Equal(t, "my task", task.Name)
	assert.Equal(t, "", task.Description)
	assert.Equal(t, "", task.Details)
	assert.Equal(t, "task", task.Type)
	assert.Equal(t, "open", task.Status)
	assert.Nil(t, task.ParentID)
}

func TestCreateTask_AllFields(t *testing.T) {
	s := newTestStore(t)

	task, err := s.CreateTask(CreateTaskParams{
		Name:        "Fix login bug",
		Description: "Safari users can't log in",
		Details:     "The session cookie SameSite attribute needs changing",
		Type:        "bug",
		Metadata: &TaskMetadata{
			CreatedBranch: "fix/login",
			CreatedDir:    "/home/user/project",
			UpdatedBranch: "fix/login",
			UpdatedDir:    "/home/user/project",
		},
	})
	require.NoError(t, err)

	assert.Equal(t, "Fix login bug", task.Name)
	assert.Equal(t, "Safari users can't log in", task.Description)
	assert.Equal(t, "The session cookie SameSite attribute needs changing", task.Details)
	assert.Equal(t, "bug", task.Type)

	meta, err := task.ParseMetadata()
	require.NoError(t, err)
	assert.Equal(t, "fix/login", meta.CreatedBranch)
	assert.Equal(t, "/home/user/project", meta.CreatedDir)
}

func TestCreateTask_InvalidType(t *testing.T) {
	s := newTestStore(t)

	_, err := s.CreateTask(CreateTaskParams{
		Name: "bad task",
		Type: "invalid",
	})
	assert.Error(t, err)
}

func TestCreateTask_WithParent(t *testing.T) {
	s := newTestStore(t)

	parent, err := s.CreateTask(CreateTaskParams{Name: "parent"})
	require.NoError(t, err)

	child, err := s.CreateTask(CreateTaskParams{
		Name:     "child",
		ParentID: &parent.ID,
	})
	require.NoError(t, err)

	assert.Equal(t, &parent.ID, child.ParentID)
}

func TestCreateTask_UniqueIDs(t *testing.T) {
	s := newTestStore(t)

	seen := make(map[string]bool)
	for i := 0; i < 50; i++ {
		task, err := s.CreateTask(CreateTaskParams{Name: "task"})
		require.NoError(t, err)
		assert.False(t, seen[task.ID], "duplicate ID: %s", task.ID)
		seen[task.ID] = true
	}
}

func TestGetTask(t *testing.T) {
	s := newTestStore(t)

	created, err := s.CreateTask(CreateTaskParams{Name: "find me"})
	require.NoError(t, err)

	found, err := s.GetTask(created.ID)
	require.NoError(t, err)
	assert.Equal(t, created.ID, found.ID)
	assert.Equal(t, "find me", found.Name)
}

func TestGetTask_NotFound(t *testing.T) {
	s := newTestStore(t)

	_, err := s.GetTask("zzzz")
	assert.Error(t, err)
}

func TestListTasks(t *testing.T) {
	s := newTestStore(t)

	s.CreateTask(CreateTaskParams{Name: "task 1"})
	s.CreateTask(CreateTaskParams{Name: "task 2"})
	s.CreateTask(CreateTaskParams{Name: "task 3"})

	tasks, err := s.ListTasks("")
	require.NoError(t, err)
	assert.Len(t, tasks, 3)
}

func TestListTasks_FilterByStatus(t *testing.T) {
	s := newTestStore(t)

	t1, _ := s.CreateTask(CreateTaskParams{Name: "open task"})
	t2, _ := s.CreateTask(CreateTaskParams{Name: "in progress task"})
	s.CreateTask(CreateTaskParams{Name: "another open task"})

	s.UpdateTaskStatus(t1.ID, "open")
	s.UpdateTaskStatus(t2.ID, "in_progress")

	tasks, err := s.ListTasks("in_progress")
	require.NoError(t, err)
	assert.Len(t, tasks, 1)
	assert.Equal(t, "in progress task", tasks[0].Name)
}

func TestListTasksFiltered(t *testing.T) {
	s := newTestStore(t)

	t1, _ := s.CreateTask(CreateTaskParams{Name: "open bug", Type: "bug"})
	s.CreateTask(CreateTaskParams{Name: "open feature", Type: "feature"})
	t3, _ := s.CreateTask(CreateTaskParams{Name: "done task"})
	s.UpdateTaskStatus(t3.ID, "done")
	s.AddTag(t1.ID, "urgent")

	// Default: hides done/cancelled.
	tasks, err := s.ListTasksFiltered(TaskFilter{})
	require.NoError(t, err)
	assert.Len(t, tasks, 2)

	// All: includes done.
	tasks, err = s.ListTasksFiltered(TaskFilter{All: true})
	require.NoError(t, err)
	assert.Len(t, tasks, 3)

	// Filter by type.
	tasks, err = s.ListTasksFiltered(TaskFilter{Type: "bug"})
	require.NoError(t, err)
	assert.Len(t, tasks, 1)
	assert.Equal(t, "open bug", tasks[0].Name)

	// Filter by status.
	tasks, err = s.ListTasksFiltered(TaskFilter{Status: "done"})
	require.NoError(t, err)
	assert.Len(t, tasks, 1)
	assert.Equal(t, "done task", tasks[0].Name)

	// Filter by tag.
	tasks, err = s.ListTasksFiltered(TaskFilter{Tag: "urgent"})
	require.NoError(t, err)
	assert.Len(t, tasks, 1)
	assert.Equal(t, "open bug", tasks[0].Name)
}

func TestUpdateTask(t *testing.T) {
	s := newTestStore(t)

	task, _ := s.CreateTask(CreateTaskParams{Name: "original", Description: "old desc", Type: "bug"})

	newName := "renamed"
	newDesc := "new desc"
	updated, err := s.UpdateTask(task.ID, UpdateTaskParams{
		Name:        &newName,
		Description: &newDesc,
	})
	require.NoError(t, err)
	assert.Equal(t, "renamed", updated.Name)
	assert.Equal(t, "new desc", updated.Description)
	assert.Equal(t, "bug", updated.Type) // unchanged
}

func TestUpdateTask_NoFields(t *testing.T) {
	s := newTestStore(t)

	task, _ := s.CreateTask(CreateTaskParams{Name: "unchanged"})

	updated, err := s.UpdateTask(task.ID, UpdateTaskParams{})
	require.NoError(t, err)
	assert.Equal(t, "unchanged", updated.Name)
}

func TestUpdateTaskStatus(t *testing.T) {
	s := newTestStore(t)

	task, _ := s.CreateTask(CreateTaskParams{Name: "do something"})
	assert.Equal(t, "open", task.Status)

	updated, err := s.UpdateTaskStatus(task.ID, "in_progress")
	require.NoError(t, err)
	assert.Equal(t, "in_progress", updated.Status)

	// Verify the task persisted correctly by re-fetching.
	refetched, err := s.GetTask(task.ID)
	require.NoError(t, err)
	assert.Equal(t, "in_progress", refetched.Status)
}

func TestUpdateTaskMetadata(t *testing.T) {
	s := newTestStore(t)

	task, _ := s.CreateTask(CreateTaskParams{Name: "meta task"})

	updated, err := s.UpdateTaskMetadata(task.ID, &TaskMetadata{
		CreatedBranch: "main",
		UpdatedBranch: "feature/x",
	})
	require.NoError(t, err)

	meta, err := updated.ParseMetadata()
	require.NoError(t, err)
	assert.Equal(t, "main", meta.CreatedBranch)
	assert.Equal(t, "feature/x", meta.UpdatedBranch)
}

func TestReadyTasks(t *testing.T) {
	s := newTestStore(t)

	t1, _ := s.CreateTask(CreateTaskParams{Name: "blocked task"})
	t2, _ := s.CreateTask(CreateTaskParams{Name: "blocking task"})
	s.CreateTask(CreateTaskParams{Name: "free task"})

	s.AddDependency(t1.ID, t2.ID)

	ready, err := s.ReadyTasks()
	require.NoError(t, err)
	assert.Len(t, ready, 2)

	names := []string{ready[0].Name, ready[1].Name}
	assert.Contains(t, names, "blocking task")
	assert.Contains(t, names, "free task")
	assert.NotContains(t, names, "blocked task")
}

func TestReadyTasks_DependencyDone(t *testing.T) {
	s := newTestStore(t)

	t1, _ := s.CreateTask(CreateTaskParams{Name: "was blocked"})
	t2, _ := s.CreateTask(CreateTaskParams{Name: "blocker"})
	s.AddDependency(t1.ID, t2.ID)

	// Complete the blocker — blocked task should now be ready.
	s.UpdateTaskStatus(t2.ID, "done")

	ready, err := s.ReadyTasks()
	require.NoError(t, err)
	assert.Len(t, ready, 1)
	assert.Equal(t, "was blocked", ready[0].Name)
}

func TestBlockedTasks(t *testing.T) {
	s := newTestStore(t)

	t1, _ := s.CreateTask(CreateTaskParams{Name: "blocked task"})
	t2, _ := s.CreateTask(CreateTaskParams{Name: "blocker"})
	s.CreateTask(CreateTaskParams{Name: "free task"})

	s.AddDependency(t1.ID, t2.ID)

	blocked, err := s.BlockedTasks()
	require.NoError(t, err)
	assert.Len(t, blocked, 1)
	assert.Equal(t, "blocked task", blocked[0].Name)
}

func TestBlockedTasks_ResolvedDep(t *testing.T) {
	s := newTestStore(t)

	t1, _ := s.CreateTask(CreateTaskParams{Name: "was blocked"})
	t2, _ := s.CreateTask(CreateTaskParams{Name: "blocker"})
	s.AddDependency(t1.ID, t2.ID)
	s.UpdateTaskStatus(t2.ID, "done")

	blocked, err := s.BlockedTasks()
	require.NoError(t, err)
	assert.Len(t, blocked, 0)
}

func TestSearchTasks(t *testing.T) {
	s := newTestStore(t)

	s.CreateTask(CreateTaskParams{Name: "Fix login bug", Description: "Safari issue"})
	s.CreateTask(CreateTaskParams{Name: "Add logout button"})
	s.CreateTask(CreateTaskParams{Name: "Update docs", Details: "Include login instructions"})

	// Search by name.
	results, err := s.SearchTasks("login")
	require.NoError(t, err)
	assert.Len(t, results, 2) // "Fix login bug" + "Update docs" (details match)

	// Search by description.
	results, err = s.SearchTasks("Safari")
	require.NoError(t, err)
	assert.Len(t, results, 1)
	assert.Equal(t, "Fix login bug", results[0].Name)

	// No results.
	results, err = s.SearchTasks("nonexistent")
	require.NoError(t, err)
	assert.Len(t, results, 0)
}

func TestSetParent(t *testing.T) {
	s := newTestStore(t)

	parent, _ := s.CreateTask(CreateTaskParams{Name: "parent"})
	child, _ := s.CreateTask(CreateTaskParams{Name: "child"})

	updated, err := s.SetParent(child.ID, &parent.ID)
	require.NoError(t, err)
	assert.Equal(t, &parent.ID, updated.ParentID)

	// Clear parent.
	cleared, err := s.SetParent(child.ID, nil)
	require.NoError(t, err)
	assert.Nil(t, cleared.ParentID)
}

func TestChildTasks(t *testing.T) {
	s := newTestStore(t)

	parent, _ := s.CreateTask(CreateTaskParams{Name: "parent"})
	c1, _ := s.CreateTask(CreateTaskParams{Name: "child 1"})
	c2, _ := s.CreateTask(CreateTaskParams{Name: "child 2"})
	s.CreateTask(CreateTaskParams{Name: "unrelated"})

	s.SetParent(c1.ID, &parent.ID)
	s.SetParent(c2.ID, &parent.ID)

	children, err := s.ChildTasks(parent.ID)
	require.NoError(t, err)
	assert.Len(t, children, 2)
	assert.Equal(t, "child 1", children[0].Name)
	assert.Equal(t, "child 2", children[1].Name)
}

func TestChildTasks_Empty(t *testing.T) {
	s := newTestStore(t)

	task, _ := s.CreateTask(CreateTaskParams{Name: "no children"})

	children, err := s.ChildTasks(task.ID)
	require.NoError(t, err)
	assert.Len(t, children, 0)
}

func TestAncestorChain(t *testing.T) {
	s := newTestStore(t)

	grandparent, _ := s.CreateTask(CreateTaskParams{Name: "grandparent"})
	parent, _ := s.CreateTask(CreateTaskParams{Name: "parent"})
	child, _ := s.CreateTask(CreateTaskParams{Name: "child"})

	s.SetParent(parent.ID, &grandparent.ID)
	s.SetParent(child.ID, &parent.ID)

	chain, err := s.AncestorChain(child.ID)
	require.NoError(t, err)
	assert.Len(t, chain, 2)
	assert.Equal(t, "parent", chain[0].Name)
	assert.Equal(t, "grandparent", chain[1].Name)
}

func TestAncestorChain_Root(t *testing.T) {
	s := newTestStore(t)

	task, _ := s.CreateTask(CreateTaskParams{Name: "root"})

	chain, err := s.AncestorChain(task.ID)
	require.NoError(t, err)
	assert.Len(t, chain, 0)
}

func TestChildCounts(t *testing.T) {
	s := newTestStore(t)

	parent, _ := s.CreateTask(CreateTaskParams{Name: "parent"})
	c1, _ := s.CreateTask(CreateTaskParams{Name: "child 1"})
	c2, _ := s.CreateTask(CreateTaskParams{Name: "child 2"})
	loner, _ := s.CreateTask(CreateTaskParams{Name: "loner"})

	s.SetParent(c1.ID, &parent.ID)
	s.SetParent(c2.ID, &parent.ID)

	counts, err := s.ChildCounts([]string{parent.ID, loner.ID})
	require.NoError(t, err)
	assert.Equal(t, 2, counts[parent.ID])
	assert.Equal(t, 0, counts[loner.ID])
}

func TestUpdateTask_DetailsAndType(t *testing.T) {
	s := newTestStore(t)

	task, _ := s.CreateTask(CreateTaskParams{Name: "original", Type: "bug"})

	newDetails := "detailed notes"
	newType := "feature"
	updated, err := s.UpdateTask(task.ID, UpdateTaskParams{
		Details: &newDetails,
		Type:    &newType,
	})
	require.NoError(t, err)
	assert.Equal(t, "original", updated.Name)   // unchanged
	assert.Equal(t, "detailed notes", updated.Details)
	assert.Equal(t, "feature", updated.Type)
}

func TestListTasksFiltered_CombinedFilters(t *testing.T) {
	s := newTestStore(t)

	t1, _ := s.CreateTask(CreateTaskParams{Name: "urgent bug", Type: "bug"})
	s.CreateTask(CreateTaskParams{Name: "normal bug", Type: "bug"})
	s.CreateTask(CreateTaskParams{Name: "urgent feature", Type: "feature"})

	s.AddTag(t1.ID, "urgent")

	// Filter by type + tag.
	tasks, err := s.ListTasksFiltered(TaskFilter{Type: "bug", Tag: "urgent"})
	require.NoError(t, err)
	assert.Len(t, tasks, 1)
	assert.Equal(t, "urgent bug", tasks[0].Name)
}

func TestListTasksFiltered_StatusAndType(t *testing.T) {
	s := newTestStore(t)

	t1, _ := s.CreateTask(CreateTaskParams{Name: "open bug", Type: "bug"})
	t2, _ := s.CreateTask(CreateTaskParams{Name: "done bug", Type: "bug"})
	s.CreateTask(CreateTaskParams{Name: "open feature", Type: "feature"})

	_ = t1
	s.UpdateTaskStatus(t2.ID, "done")

	tasks, err := s.ListTasksFiltered(TaskFilter{Type: "bug", Status: "done"})
	require.NoError(t, err)
	assert.Len(t, tasks, 1)
	assert.Equal(t, "done bug", tasks[0].Name)
}

func TestDependenciesOf_Empty(t *testing.T) {
	s := newTestStore(t)

	task, _ := s.CreateTask(CreateTaskParams{Name: "no deps"})

	deps, err := s.DependenciesOf(task.ID)
	require.NoError(t, err)
	assert.Len(t, deps, 0)
}

func TestCommentsForTask_Empty(t *testing.T) {
	s := newTestStore(t)

	task, _ := s.CreateTask(CreateTaskParams{Name: "no comments"})

	comments, err := s.CommentsForTask(task.ID)
	require.NoError(t, err)
	assert.Len(t, comments, 0)
}

func TestParseMetadata_NoMetadata(t *testing.T) {
	s := newTestStore(t)

	task, _ := s.CreateTask(CreateTaskParams{Name: "no metadata"})

	meta, err := task.ParseMetadata()
	require.NoError(t, err)
	// No metadata set — returns zero-value struct (not nil).
	assert.Empty(t, meta.CreatedBranch)
	assert.Empty(t, meta.CreatedDir)
}

func TestListTasksFiltered_RootOnly(t *testing.T) {
	s := newTestStore(t)

	parent, _ := s.CreateTask(CreateTaskParams{Name: "root task"})
	child, _ := s.CreateTask(CreateTaskParams{Name: "child task"})
	s.SetParent(child.ID, &parent.ID)

	tasks, err := s.ListTasksFiltered(TaskFilter{RootOnly: true})
	require.NoError(t, err)
	assert.Len(t, tasks, 1)
	assert.Equal(t, "root task", tasks[0].Name)
}
