package store

import (
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestAddComment(t *testing.T) {
	s := newTestStore(t)

	task, _ := s.CreateTask(CreateTaskParams{Name: "task"})

	comment, err := s.AddComment(task.ID, "comment", "work in progress")
	require.NoError(t, err)

	assert.Equal(t, task.ID, comment.TaskID)
	assert.Equal(t, "comment", comment.Type)
	assert.Equal(t, "work in progress", comment.Content)
}

func TestAddComment_DefaultType(t *testing.T) {
	s := newTestStore(t)

	task, _ := s.CreateTask(CreateTaskParams{Name: "task"})

	comment, err := s.AddComment(task.ID, "", "some note")
	require.NoError(t, err)
	assert.Equal(t, "comment", comment.Type)
}

func TestCommentsForTask(t *testing.T) {
	s := newTestStore(t)

	task, _ := s.CreateTask(CreateTaskParams{Name: "task"})

	s.AddComment(task.ID, "comment", "first")
	s.AddComment(task.ID, "comment", "second")
	s.AddComment(task.ID, "close", "done with this")

	comments, err := s.CommentsForTask(task.ID)
	require.NoError(t, err)
	assert.Len(t, comments, 3)
}

func TestEventComments(t *testing.T) {
	s := newTestStore(t)

	task, _ := s.CreateTask(CreateTaskParams{Name: "task"})

	s.AddComment(task.ID, "comment", "regular note")
	s.AddComment(task.ID, "close", "completed the work")
	s.AddComment(task.ID, "reopen", "actually not done")

	events, err := s.EventComments(task.ID)
	require.NoError(t, err)
	assert.Len(t, events, 2)
	assert.Equal(t, "close", events[0].Type)
	assert.Equal(t, "reopen", events[1].Type)
}

func TestLatestCloseReason(t *testing.T) {
	s := newTestStore(t)

	task, _ := s.CreateTask(CreateTaskParams{Name: "task"})

	s.AddComment(task.ID, "close", "first close")
	s.AddComment(task.ID, "reopen", "reopened")
	s.AddComment(task.ID, "close", "final close")

	reason, err := s.LatestCloseReason(task.ID)
	require.NoError(t, err)
	assert.Equal(t, "final close", reason.Content)
}

func TestDeleteComment(t *testing.T) {
	s := newTestStore(t)

	task, _ := s.CreateTask(CreateTaskParams{Name: "task"})
	c, _ := s.AddComment(task.ID, "comment", "delete me")

	err := s.DeleteComment(c.ID)
	require.NoError(t, err)

	comments, _ := s.CommentsForTask(task.ID)
	assert.Len(t, comments, 0)
}

func TestDeleteComment_NotFound(t *testing.T) {
	s := newTestStore(t)

	err := s.DeleteComment(9999)
	assert.Error(t, err)
}

func TestLatestCloseReason_NeverClosed(t *testing.T) {
	s := newTestStore(t)

	task, _ := s.CreateTask(CreateTaskParams{Name: "task"})

	_, err := s.LatestCloseReason(task.ID)
	assert.Error(t, err)
}
