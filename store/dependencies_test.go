package store

import (
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestAddDependency(t *testing.T) {
	s := newTestStore(t)

	t1, _ := s.CreateTask(CreateTaskParams{Name: "task A"})
	t2, _ := s.CreateTask(CreateTaskParams{Name: "task B"})

	err := s.AddDependency(t1.ID, t2.ID)
	assert.NoError(t, err)

	deps, err := s.DependenciesOf(t1.ID)
	require.NoError(t, err)
	assert.Len(t, deps, 1)
	assert.Equal(t, t2.ID, deps[0].DependsOnID)
}

func TestAddDependency_SelfReference(t *testing.T) {
	s := newTestStore(t)

	task, _ := s.CreateTask(CreateTaskParams{Name: "task"})

	err := s.AddDependency(task.ID, task.ID)
	assert.Error(t, err)
}

func TestAddDependency_Duplicate(t *testing.T) {
	s := newTestStore(t)

	t1, _ := s.CreateTask(CreateTaskParams{Name: "task A"})
	t2, _ := s.CreateTask(CreateTaskParams{Name: "task B"})

	err := s.AddDependency(t1.ID, t2.ID)
	require.NoError(t, err)

	err = s.AddDependency(t1.ID, t2.ID)
	assert.Error(t, err)
}

func TestRemoveDependency(t *testing.T) {
	s := newTestStore(t)

	t1, _ := s.CreateTask(CreateTaskParams{Name: "task A"})
	t2, _ := s.CreateTask(CreateTaskParams{Name: "task B"})

	s.AddDependency(t1.ID, t2.ID)

	err := s.RemoveDependency(t1.ID, t2.ID)
	assert.NoError(t, err)

	deps, err := s.DependenciesOf(t1.ID)
	require.NoError(t, err)
	assert.Len(t, deps, 0)
}

func TestRemoveDependency_NotFound(t *testing.T) {
	s := newTestStore(t)

	err := s.RemoveDependency("zzzz", "yyyy")
	assert.Error(t, err)
}

func TestDependentsOf(t *testing.T) {
	s := newTestStore(t)

	blocker, _ := s.CreateTask(CreateTaskParams{Name: "blocker"})
	blocked1, _ := s.CreateTask(CreateTaskParams{Name: "blocked 1"})
	blocked2, _ := s.CreateTask(CreateTaskParams{Name: "blocked 2"})

	s.AddDependency(blocked1.ID, blocker.ID)
	s.AddDependency(blocked2.ID, blocker.ID)

	dependents, err := s.DependentsOf(blocker.ID)
	require.NoError(t, err)
	assert.Len(t, dependents, 2)
}

func TestDependentsOf_Empty(t *testing.T) {
	s := newTestStore(t)

	task, _ := s.CreateTask(CreateTaskParams{Name: "no dependents"})

	dependents, err := s.DependentsOf(task.ID)
	require.NoError(t, err)
	assert.Len(t, dependents, 0)
}
