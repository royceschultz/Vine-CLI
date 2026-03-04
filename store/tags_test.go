package store

import (
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestAddTag(t *testing.T) {
	s := newTestStore(t)

	task, _ := s.CreateTask(CreateTaskParams{Name: "task"})

	err := s.AddTag(task.ID, "frontend")
	assert.NoError(t, err)

	tags, err := s.TagsForTask(task.ID)
	require.NoError(t, err)
	assert.Len(t, tags, 1)
	assert.Equal(t, "frontend", tags[0].Name)
}

func TestAddTag_Multiple(t *testing.T) {
	s := newTestStore(t)

	task, _ := s.CreateTask(CreateTaskParams{Name: "task"})

	s.AddTag(task.ID, "frontend")
	s.AddTag(task.ID, "urgent")
	s.AddTag(task.ID, "bug")

	tags, err := s.TagsForTask(task.ID)
	require.NoError(t, err)
	assert.Len(t, tags, 3)
}

func TestAddTag_Duplicate(t *testing.T) {
	s := newTestStore(t)

	task, _ := s.CreateTask(CreateTaskParams{Name: "task"})

	err := s.AddTag(task.ID, "frontend")
	require.NoError(t, err)

	// Adding same tag again should not error (ON CONFLICT DO NOTHING).
	err = s.AddTag(task.ID, "frontend")
	assert.NoError(t, err)

	tags, err := s.TagsForTask(task.ID)
	require.NoError(t, err)
	assert.Len(t, tags, 1)
}

func TestAddTag_SharedAcrossTasks(t *testing.T) {
	s := newTestStore(t)

	t1, _ := s.CreateTask(CreateTaskParams{Name: "task 1"})
	t2, _ := s.CreateTask(CreateTaskParams{Name: "task 2"})

	s.AddTag(t1.ID, "shared")
	s.AddTag(t2.ID, "shared")

	tasks, err := s.TasksByTag("shared")
	require.NoError(t, err)
	assert.Len(t, tasks, 2)
}

func TestRemoveTag(t *testing.T) {
	s := newTestStore(t)

	task, _ := s.CreateTask(CreateTaskParams{Name: "task"})

	s.AddTag(task.ID, "remove-me")

	err := s.RemoveTag(task.ID, "remove-me")
	assert.NoError(t, err)

	tags, err := s.TagsForTask(task.ID)
	require.NoError(t, err)
	assert.Len(t, tags, 0)
}

func TestRemoveTag_NotFound(t *testing.T) {
	s := newTestStore(t)

	task, _ := s.CreateTask(CreateTaskParams{Name: "task"})

	err := s.RemoveTag(task.ID, "nonexistent")
	assert.Error(t, err)
}

func TestListTags(t *testing.T) {
	s := newTestStore(t)

	t1, _ := s.CreateTask(CreateTaskParams{Name: "task 1"})
	t2, _ := s.CreateTask(CreateTaskParams{Name: "task 2"})

	s.AddTag(t1.ID, "frontend")
	s.AddTag(t2.ID, "frontend")
	s.AddTag(t1.ID, "urgent")

	tags, err := s.ListTags()
	require.NoError(t, err)
	assert.Len(t, tags, 2)

	// Ordered by name.
	assert.Equal(t, "frontend", tags[0].Name)
	assert.Equal(t, 2, tags[0].Count)
	assert.Equal(t, "urgent", tags[1].Name)
	assert.Equal(t, 1, tags[1].Count)
}

func TestListTags_IncludesOrphans(t *testing.T) {
	s := newTestStore(t)

	task, _ := s.CreateTask(CreateTaskParams{Name: "task"})
	s.AddTag(task.ID, "will-be-orphan")
	s.RemoveTag(task.ID, "will-be-orphan")

	tags, err := s.ListTags()
	require.NoError(t, err)
	assert.Len(t, tags, 1)
	assert.Equal(t, "will-be-orphan", tags[0].Name)
	assert.Equal(t, 0, tags[0].Count)
}

func TestPruneOrphanTags(t *testing.T) {
	s := newTestStore(t)

	task, _ := s.CreateTask(CreateTaskParams{Name: "task"})
	s.AddTag(task.ID, "keep-me")
	s.AddTag(task.ID, "orphan")
	s.RemoveTag(task.ID, "orphan")

	pruned, err := s.PruneOrphanTags()
	require.NoError(t, err)
	assert.Equal(t, 1, pruned)

	// Only "keep-me" remains.
	tags, err := s.ListTags()
	require.NoError(t, err)
	assert.Len(t, tags, 1)
	assert.Equal(t, "keep-me", tags[0].Name)
}

func TestPruneOrphanTags_NoneToRemove(t *testing.T) {
	s := newTestStore(t)

	pruned, err := s.PruneOrphanTags()
	require.NoError(t, err)
	assert.Equal(t, 0, pruned)
}

func TestTasksByTag(t *testing.T) {
	s := newTestStore(t)

	t1, _ := s.CreateTask(CreateTaskParams{Name: "frontend task"})
	s.CreateTask(CreateTaskParams{Name: "backend task"})
	t3, _ := s.CreateTask(CreateTaskParams{Name: "another frontend task"})

	s.AddTag(t1.ID, "frontend")
	s.AddTag(t3.ID, "frontend")

	tasks, err := s.TasksByTag("frontend")
	require.NoError(t, err)
	assert.Len(t, tasks, 2)
	assert.Equal(t, "frontend task", tasks[0].Name)
	assert.Equal(t, "another frontend task", tasks[1].Name)
}
