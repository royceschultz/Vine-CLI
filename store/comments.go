package store

import (
	"fmt"
	"time"
)

type Comment struct {
	ID        int64  `db:"id"         json:"id"`
	TaskID    string `db:"task_id"    json:"task_id"`
	Type      string `db:"type"       json:"type"`
	Content   string `db:"content"    json:"content"`
	Metadata  string `db:"metadata"   json:"metadata"`
	CreatedAt string `db:"created_at" json:"created_at"`
}

func (s *Store) AddComment(taskID string, commentType, content string) (*Comment, error) {
	now := time.Now().UTC().Format(time.RFC3339)

	if commentType == "" {
		commentType = "comment"
	}

	result, err := s.db.Exec(
		`INSERT INTO comments (task_id, type, content, created_at)
		 VALUES (?, ?, ?, ?)`,
		taskID, commentType, content, now,
	)
	if err != nil {
		return nil, fmt.Errorf("adding comment to task %s: %w", taskID, err)
	}
	id, _ := result.LastInsertId()

	var c Comment
	if err := s.db.Get(&c, "SELECT * FROM comments WHERE id = ?", id); err != nil {
		return nil, fmt.Errorf("getting comment %d: %w", id, err)
	}
	return &c, nil
}

func (s *Store) CommentsForTask(taskID string) ([]Comment, error) {
	var comments []Comment
	err := s.db.Select(&comments,
		"SELECT * FROM comments WHERE task_id = ? ORDER BY created_at", taskID)
	if err != nil {
		return nil, fmt.Errorf("listing comments for task %s: %w", taskID, err)
	}
	return comments, nil
}

func (s *Store) EventComments(taskID string) ([]Comment, error) {
	var comments []Comment
	err := s.db.Select(&comments,
		"SELECT * FROM comments WHERE task_id = ? AND type != 'comment' ORDER BY created_at",
		taskID)
	if err != nil {
		return nil, fmt.Errorf("listing event comments for task %s: %w", taskID, err)
	}
	return comments, nil
}

func (s *Store) DeleteComment(id int64) error {
	result, err := s.db.Exec("DELETE FROM comments WHERE id = ?", id)
	if err != nil {
		return fmt.Errorf("deleting comment %d: %w", id, err)
	}
	n, _ := result.RowsAffected()
	if n == 0 {
		return fmt.Errorf("comment %d not found", id)
	}
	return nil
}

func (s *Store) LatestCloseReason(taskID string) (*Comment, error) {
	var c Comment
	err := s.db.Get(&c,
		"SELECT * FROM comments WHERE task_id = ? AND type = 'close' ORDER BY id DESC LIMIT 1",
		taskID)
	if err != nil {
		return nil, fmt.Errorf("getting close reason for task %s: %w", taskID, err)
	}
	return &c, nil
}
