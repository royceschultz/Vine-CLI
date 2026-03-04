package store

import "fmt"

type Tag struct {
	ID   int64  `db:"id"   json:"id"`
	Name string `db:"name" json:"name"`
}

func (s *Store) AddTag(taskID string, tagName string) error {
	// Upsert the tag.
	_, err := s.db.Exec(
		"INSERT INTO tags (name) VALUES (?) ON CONFLICT (name) DO NOTHING",
		tagName,
	)
	if err != nil {
		return fmt.Errorf("creating tag %q: %w", tagName, err)
	}

	// Get the tag ID.
	var tag Tag
	if err := s.db.Get(&tag, "SELECT * FROM tags WHERE name = ?", tagName); err != nil {
		return fmt.Errorf("getting tag %q: %w", tagName, err)
	}

	// Link to task.
	_, err = s.db.Exec(
		"INSERT INTO task_tags (task_id, tag_id) VALUES (?, ?) ON CONFLICT DO NOTHING",
		taskID, tag.ID,
	)
	if err != nil {
		return fmt.Errorf("tagging task %s with %q: %w", taskID, tagName, err)
	}
	return nil
}

func (s *Store) RemoveTag(taskID string, tagName string) error {
	result, err := s.db.Exec(
		`DELETE FROM task_tags WHERE task_id = ? AND tag_id = (
			SELECT id FROM tags WHERE name = ?
		)`,
		taskID, tagName,
	)
	if err != nil {
		return fmt.Errorf("removing tag %q from task %s: %w", tagName, taskID, err)
	}
	n, _ := result.RowsAffected()
	if n == 0 {
		return fmt.Errorf("tag %q not found on task %s", tagName, taskID)
	}
	return nil
}

func (s *Store) TagsForTask(taskID string) ([]Tag, error) {
	var tags []Tag
	err := s.db.Select(&tags,
		`SELECT t.* FROM tags t
		 JOIN task_tags tt ON tt.tag_id = t.id
		 WHERE tt.task_id = ?
		 ORDER BY t.name`,
		taskID)
	if err != nil {
		return nil, fmt.Errorf("listing tags for task %s: %w", taskID, err)
	}
	return tags, nil
}

// TagWithCount holds a tag name and its associated task count.
type TagWithCount struct {
	Name  string `db:"name"  json:"name"`
	Count int    `db:"count" json:"count"`
}

func (s *Store) ListTags() ([]TagWithCount, error) {
	var tags []TagWithCount
	err := s.db.Select(&tags,
		`SELECT t.name, COUNT(tt.task_id) AS count
		 FROM tags t
		 LEFT JOIN task_tags tt ON tt.tag_id = t.id
		 GROUP BY t.id, t.name
		 ORDER BY t.name`)
	if err != nil {
		return nil, fmt.Errorf("listing tags: %w", err)
	}
	return tags, nil
}

func (s *Store) PruneOrphanTags() (int, error) {
	result, err := s.db.Exec(
		`DELETE FROM tags WHERE id NOT IN (
			SELECT DISTINCT tag_id FROM task_tags
		)`)
	if err != nil {
		return 0, fmt.Errorf("pruning orphan tags: %w", err)
	}
	n, _ := result.RowsAffected()
	return int(n), nil
}

func (s *Store) TasksByTag(tagName string) ([]Task, error) {
	var tasks []Task
	err := s.db.Select(&tasks,
		`SELECT t.* FROM tasks t
		 JOIN task_tags tt ON tt.task_id = t.id
		 JOIN tags tg ON tg.id = tt.tag_id
		 WHERE tg.name = ?
		 ORDER BY t.created_at`,
		tagName)
	if err != nil {
		return nil, fmt.Errorf("listing tasks with tag %q: %w", tagName, err)
	}
	return tasks, nil
}
