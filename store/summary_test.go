package store

import (
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestTaskSummary_Empty(t *testing.T) {
	s := newTestStore(t)

	counts, err := s.TaskSummary()
	require.NoError(t, err)
	assert.Len(t, counts, 0)
}

func TestTaskSummary_Basic(t *testing.T) {
	s := newTestStore(t)

	s.CreateTask(CreateTaskParams{Name: "task 1"})
	s.CreateTask(CreateTaskParams{Name: "task 2"})
	t3, _ := s.CreateTask(CreateTaskParams{Name: "task 3"})
	s.UpdateTaskStatus(t3.ID, "in_progress")

	counts, err := s.TaskSummary()
	require.NoError(t, err)

	byStatus := make(map[string]int)
	for _, c := range counts {
		byStatus[c.Status] = c.Count
	}
	assert.Equal(t, 2, byStatus["ready"])
	assert.Equal(t, 1, byStatus["in_progress"])
}

func TestTaskSummary_BlockedVsReady(t *testing.T) {
	s := newTestStore(t)

	blocker, _ := s.CreateTask(CreateTaskParams{Name: "blocker"})
	blocked, _ := s.CreateTask(CreateTaskParams{Name: "blocked"})
	s.CreateTask(CreateTaskParams{Name: "free"})
	s.AddDependency(blocked.ID, blocker.ID)

	counts, err := s.TaskSummary()
	require.NoError(t, err)

	byStatus := make(map[string]int)
	for _, c := range counts {
		byStatus[c.Status] = c.Count
	}
	assert.Equal(t, 2, byStatus["ready"])
	assert.Equal(t, 1, byStatus["blocked"])
}

func TestTaskSummary_BlockedBecomeReady(t *testing.T) {
	s := newTestStore(t)

	blocker, _ := s.CreateTask(CreateTaskParams{Name: "blocker"})
	blocked, _ := s.CreateTask(CreateTaskParams{Name: "was blocked"})
	s.AddDependency(blocked.ID, blocker.ID)

	s.UpdateTaskStatus(blocker.ID, "done")

	counts, err := s.TaskSummary()
	require.NoError(t, err)

	byStatus := make(map[string]int)
	for _, c := range counts {
		byStatus[c.Status] = c.Count
	}
	assert.Equal(t, 1, byStatus["ready"])
	assert.Equal(t, 0, byStatus["blocked"])
	assert.Equal(t, 1, byStatus["done"])
}

func TestTaskSummaryDetailed(t *testing.T) {
	s := newTestStore(t)

	s.CreateTask(CreateTaskParams{Name: "bug 1", Type: "bug"})
	s.CreateTask(CreateTaskParams{Name: "bug 2", Type: "bug"})
	s.CreateTask(CreateTaskParams{Name: "feature 1", Type: "feature"})
	t4, _ := s.CreateTask(CreateTaskParams{Name: "done task", Type: "task"})
	s.UpdateTaskStatus(t4.ID, "done")

	counts, err := s.TaskSummaryDetailed()
	require.NoError(t, err)

	// Build a lookup: status+type -> count
	lookup := make(map[string]int)
	for _, c := range counts {
		lookup[c.Status+":"+c.Type] = c.Count
	}
	assert.Equal(t, 2, lookup["ready:bug"])
	assert.Equal(t, 1, lookup["ready:feature"])
	assert.Equal(t, 1, lookup["done:task"])
}

func TestTaskSummary_StatusOrder(t *testing.T) {
	s := newTestStore(t)

	t1, _ := s.CreateTask(CreateTaskParams{Name: "ready task"})
	t2, _ := s.CreateTask(CreateTaskParams{Name: "in progress"})
	t3, _ := s.CreateTask(CreateTaskParams{Name: "done"})
	t4, _ := s.CreateTask(CreateTaskParams{Name: "blocked"})
	blocker, _ := s.CreateTask(CreateTaskParams{Name: "blocker"})

	_ = t1
	s.UpdateTaskStatus(t2.ID, "in_progress")
	s.UpdateTaskStatus(t3.ID, "done")
	s.AddDependency(t4.ID, blocker.ID)

	counts, err := s.TaskSummary()
	require.NoError(t, err)

	// Verify ordering: ready, blocked, in_progress, done
	statuses := make([]string, len(counts))
	for i, c := range counts {
		statuses[i] = c.Status
	}
	assert.Equal(t, []string{"ready", "blocked", "in_progress", "done"}, statuses)
}
