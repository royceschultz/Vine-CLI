package store

import (
	"encoding/json"
	"fmt"
	"os"
	"strings"
	"time"

	"github.com/jmoiron/sqlx"

	"vine/utils"
)

type TaskMetadata struct {
	CreatedBranch string `json:"created_branch,omitempty"`
	CreatedDir    string `json:"created_dir,omitempty"`
	CreatedBy     string `json:"created_by,omitempty"`
	UpdatedBranch string `json:"updated_branch,omitempty"`
	UpdatedDir    string `json:"updated_dir,omitempty"`
	UpdatedBy     string `json:"updated_by,omitempty"`
}

type Task struct {
	ID          string  `db:"id"          json:"id"`
	Name        string  `db:"name"        json:"name"`
	Description string  `db:"description" json:"description"`
	Details     string  `db:"details"     json:"details,omitempty"`
	Type        string  `db:"type"        json:"type"`
	Status      string  `db:"status"      json:"status"`
	ParentID    *string `db:"parent_id"   json:"parent_id,omitempty"`
	Metadata    string  `db:"metadata"    json:"metadata"`
	CreatedAt   string  `db:"created_at"  json:"created_at"`
	UpdatedAt   string  `db:"updated_at"  json:"updated_at"`
}

// ParseMetadata decodes the JSON metadata field into a TaskMetadata struct.
func (t *Task) ParseMetadata() (*TaskMetadata, error) {
	var m TaskMetadata
	if err := json.Unmarshal([]byte(t.Metadata), &m); err != nil {
		return nil, fmt.Errorf("parsing task metadata: %w", err)
	}
	return &m, nil
}

// TaskWithDeps wraps a Task with dependency ID arrays for list output.
type TaskWithDeps struct {
	Task
	DependsOnIDs []string `json:"depends_on_ids"`
	BlocksIDs    []string `json:"blocks_ids"`
}

type CreateTaskParams struct {
	Name        string
	Description string
	Details     string
	Type        string
	ParentID    *string
	Metadata    *TaskMetadata
}

const maxIDRetries = 10

func (s *Store) CreateTask(p CreateTaskParams) (*Task, error) {
	now := time.Now().UTC().Format(time.RFC3339)

	taskType := p.Type
	if taskType == "" {
		taskType = "task"
	}

	meta := "{}"
	if p.Metadata != nil {
		b, err := json.Marshal(p.Metadata)
		if err != nil {
			return nil, fmt.Errorf("marshaling metadata: %w", err)
		}
		meta = string(b)
	}

	// Generate a unique 4-char ID, retrying on collision.
	var id string
	for i := 0; i < maxIDRetries; i++ {
		candidate := utils.GenerateID(utils.IDLength)
		var exists int
		if err := s.db.Get(&exists, "SELECT COUNT(*) FROM tasks WHERE id = ?", candidate); err != nil {
			return nil, fmt.Errorf("checking ID collision: %w", err)
		}
		if exists == 0 {
			id = candidate
			break
		}
		fmt.Fprintf(os.Stderr, "whoa, ID collision on %q! that's mass-extinction-asteroid rare. retrying...\n", candidate)
	}
	if id == "" {
		return nil, fmt.Errorf("failed to generate unique task ID after %d attempts", maxIDRetries)
	}

	_, err := s.db.Exec(
		`INSERT INTO tasks (id, name, description, details, type, parent_id, metadata, created_at, updated_at)
		 VALUES (?, ?, ?, ?, ?, ?, ?, ?, ?)`,
		id, p.Name, p.Description, p.Details, taskType, p.ParentID, meta, now, now,
	)
	if err != nil {
		return nil, fmt.Errorf("inserting task: %w", err)
	}
	return s.GetTask(id)
}

func (s *Store) GetTask(id string) (*Task, error) {
	var t Task
	if err := s.db.Get(&t, "SELECT * FROM tasks WHERE id = ?", id); err != nil {
		return nil, fmt.Errorf("getting task %s: %w", id, err)
	}
	return &t, nil
}

func (s *Store) ListTasks(status string) ([]Task, error) {
	var tasks []Task
	var err error
	if status == "" {
		err = s.db.Select(&tasks, "SELECT * FROM tasks ORDER BY created_at")
	} else {
		err = s.db.Select(&tasks, "SELECT * FROM tasks WHERE status = ? ORDER BY created_at", status)
	}
	if err != nil {
		return nil, fmt.Errorf("listing tasks: %w", err)
	}
	return tasks, nil
}

// TaskFilter specifies criteria for listing tasks.
type TaskFilter struct {
	Status   string // exact status match
	Type     string // exact type match
	Tag      string // tag name
	All      bool   // include done/cancelled (default: hide them)
	RootOnly bool   // only tasks with no parent
}

func (s *Store) ListTasksFiltered(f TaskFilter) ([]Task, error) {
	query := "SELECT DISTINCT t.* FROM tasks t"
	var args []any

	if f.Tag != "" {
		query += " JOIN task_tags tt ON tt.task_id = t.id JOIN tags tg ON tg.id = tt.tag_id"
	}

	var where []string

	if f.Status != "" {
		where = append(where, "t.status = ?")
		args = append(args, f.Status)
	} else if !f.All {
		where = append(where, "t.status NOT IN ('done', 'cancelled')")
	}

	if f.Type != "" {
		where = append(where, "t.type = ?")
		args = append(args, f.Type)
	}

	if f.Tag != "" {
		where = append(where, "tg.name = ?")
		args = append(args, f.Tag)
	}

	if f.RootOnly {
		where = append(where, "t.parent_id IS NULL")
	}

	if len(where) > 0 {
		query += " WHERE " + strings.Join(where, " AND ")
	}

	query += " ORDER BY t.created_at"

	var tasks []Task
	if err := s.db.Select(&tasks, query, args...); err != nil {
		return nil, fmt.Errorf("listing tasks: %w", err)
	}
	return tasks, nil
}

// UpdateTaskParams specifies fields to update. Nil pointers are skipped.
type UpdateTaskParams struct {
	Name        *string
	Description *string
	Details     *string
	Type        *string
}

func (s *Store) UpdateTask(id string, p UpdateTaskParams) (*Task, error) {
	var sets []string
	var args []any

	if p.Name != nil {
		sets = append(sets, "name = ?")
		args = append(args, *p.Name)
	}
	if p.Description != nil {
		sets = append(sets, "description = ?")
		args = append(args, *p.Description)
	}
	if p.Details != nil {
		sets = append(sets, "details = ?")
		args = append(args, *p.Details)
	}
	if p.Type != nil {
		sets = append(sets, "type = ?")
		args = append(args, *p.Type)
	}

	if len(sets) == 0 {
		return s.GetTask(id)
	}

	now := time.Now().UTC().Format(time.RFC3339)
	sets = append(sets, "updated_at = ?")
	args = append(args, now)
	args = append(args, id)

	query := "UPDATE tasks SET " + strings.Join(sets, ", ") + " WHERE id = ?"
	_, err := s.db.Exec(query, args...)
	if err != nil {
		return nil, fmt.Errorf("updating task %s: %w", id, err)
	}
	return s.GetTask(id)
}

func (s *Store) UpdateTaskStatus(id string, status string) (*Task, error) {
	now := time.Now().UTC().Format(time.RFC3339)
	_, err := s.db.Exec(
		"UPDATE tasks SET status = ?, updated_at = ? WHERE id = ?",
		status, now, id,
	)
	if err != nil {
		return nil, fmt.Errorf("updating task %s status: %w", id, err)
	}
	return s.GetTask(id)
}

func (s *Store) UpdateTaskMetadata(id string, meta *TaskMetadata) (*Task, error) {
	now := time.Now().UTC().Format(time.RFC3339)
	b, err := json.Marshal(meta)
	if err != nil {
		return nil, fmt.Errorf("marshaling metadata: %w", err)
	}
	_, err = s.db.Exec(
		"UPDATE tasks SET metadata = ?, updated_at = ? WHERE id = ?",
		string(b), now, id,
	)
	if err != nil {
		return nil, fmt.Errorf("updating task %s metadata: %w", id, err)
	}
	return s.GetTask(id)
}

func (s *Store) ReadyTasks() ([]Task, error) {
	var tasks []Task
	err := s.db.Select(&tasks, `
		SELECT t.* FROM tasks t
		WHERE t.status = 'open'
		  AND NOT EXISTS (
		      SELECT 1 FROM dependencies d
		      JOIN tasks dep ON dep.id = d.depends_on_id
		      WHERE d.task_id = t.id
		        AND dep.status NOT IN ('done', 'cancelled')
		  )
		ORDER BY t.created_at
	`)
	if err != nil {
		return nil, fmt.Errorf("listing ready tasks: %w", err)
	}
	return tasks, nil
}

func (s *Store) BlockedTasks() ([]Task, error) {
	var tasks []Task
	err := s.db.Select(&tasks, `
		SELECT t.* FROM tasks t
		WHERE t.status = 'open'
		  AND EXISTS (
		      SELECT 1 FROM dependencies d
		      JOIN tasks dep ON dep.id = d.depends_on_id
		      WHERE d.task_id = t.id
		        AND dep.status NOT IN ('done', 'cancelled')
		  )
		ORDER BY t.created_at
	`)
	if err != nil {
		return nil, fmt.Errorf("listing blocked tasks: %w", err)
	}
	return tasks, nil
}

// EnrichEffectiveStatus replaces the Status field for "open" tasks with
// "blocked" (has unfulfilled dependencies) or "ready" (no unfulfilled deps).
// Tasks with other statuses are left unchanged. Modifies tasks in-place.
func (s *Store) EnrichEffectiveStatus(tasks []Task) error {
	var openIDs []string
	openIdx := map[string][]int{}
	for i, t := range tasks {
		if t.Status == "open" {
			openIDs = append(openIDs, t.ID)
			openIdx[t.ID] = append(openIdx[t.ID], i)
		}
	}
	if len(openIDs) == 0 {
		return nil
	}

	// Find which open tasks have unfulfilled dependencies.
	query, args, err := sqlx.In(`
		SELECT DISTINCT d.task_id FROM dependencies d
		JOIN tasks dep ON dep.id = d.depends_on_id
		WHERE d.task_id IN (?)
		  AND dep.status NOT IN ('done', 'cancelled')
	`, openIDs)
	if err != nil {
		return fmt.Errorf("building blocked-check query: %w", err)
	}
	query = s.db.Rebind(query)

	var blockedIDs []string
	if err := s.db.Select(&blockedIDs, query, args...); err != nil {
		return fmt.Errorf("querying blocked task IDs: %w", err)
	}

	blockedSet := make(map[string]bool, len(blockedIDs))
	for _, id := range blockedIDs {
		blockedSet[id] = true
	}

	for id, indices := range openIdx {
		status := "ready"
		if blockedSet[id] {
			status = "blocked"
		}
		for _, i := range indices {
			tasks[i].Status = status
		}
	}
	return nil
}

// EnrichEffectiveStatusPtr applies EnrichEffectiveStatus to pointer slices.
func (s *Store) EnrichEffectiveStatusPtr(tasks []*Task) error {
	if len(tasks) == 0 {
		return nil
	}
	plain := make([]Task, len(tasks))
	for i, t := range tasks {
		plain[i] = *t
	}
	if err := s.EnrichEffectiveStatus(plain); err != nil {
		return err
	}
	for i, t := range plain {
		tasks[i].Status = t.Status
	}
	return nil
}

func (s *Store) SearchTasks(query string) ([]Task, error) {
	var tasks []Task
	pattern := "%" + query + "%"
	err := s.db.Select(&tasks, `
		SELECT * FROM tasks
		WHERE name LIKE ? OR description LIKE ? OR details LIKE ?
		ORDER BY created_at
	`, pattern, pattern, pattern)
	if err != nil {
		return nil, fmt.Errorf("searching tasks: %w", err)
	}
	return tasks, nil
}

func (s *Store) SetParent(childID string, parentID *string) (*Task, error) {
	now := time.Now().UTC().Format(time.RFC3339)
	_, err := s.db.Exec(
		"UPDATE tasks SET parent_id = ?, updated_at = ? WHERE id = ?",
		parentID, now, childID,
	)
	if err != nil {
		return nil, fmt.Errorf("setting parent for task %s: %w", childID, err)
	}
	return s.GetTask(childID)
}

func (s *Store) ChildTasks(parentID string) ([]Task, error) {
	var tasks []Task
	err := s.db.Select(&tasks,
		"SELECT * FROM tasks WHERE parent_id = ? ORDER BY created_at", parentID)
	if err != nil {
		return nil, fmt.Errorf("listing children of task %s: %w", parentID, err)
	}
	return tasks, nil
}

// AncestorChain returns the parent chain from immediate parent up to the root.
func (s *Store) AncestorChain(taskID string) ([]Task, error) {
	var chain []Task
	current := taskID
	for i := 0; i < 100; i++ { // depth limit
		task, err := s.GetTask(current)
		if err != nil || task.ParentID == nil {
			break
		}
		parent, err := s.GetTask(*task.ParentID)
		if err != nil {
			break
		}
		chain = append(chain, *parent)
		current = parent.ID
	}
	return chain, nil
}

// IncompleteChildTasks returns direct children that are not done or cancelled.
func (s *Store) IncompleteChildTasks(parentID string) ([]Task, error) {
	var tasks []Task
	err := s.db.Select(&tasks,
		"SELECT * FROM tasks WHERE parent_id = ? AND status NOT IN ('done', 'cancelled') ORDER BY created_at",
		parentID)
	if err != nil {
		return nil, fmt.Errorf("listing incomplete children of task %s: %w", parentID, err)
	}
	return tasks, nil
}

// ChildCounts returns a map of task ID → number of direct children.
// Only returns entries for tasks that have at least one child.
func (s *Store) ChildCounts(taskIDs []string) (map[string]int, error) {
	if len(taskIDs) == 0 {
		return nil, nil
	}

	query, args, err := sqlx.In(
		"SELECT parent_id, COUNT(*) AS count FROM tasks WHERE parent_id IN (?) GROUP BY parent_id",
		taskIDs,
	)
	if err != nil {
		return nil, fmt.Errorf("building child counts query: %w", err)
	}
	query = s.db.Rebind(query)

	type row struct {
		ParentID string `db:"parent_id"`
		Count    int    `db:"count"`
	}
	var rows []row
	if err := s.db.Select(&rows, query, args...); err != nil {
		return nil, fmt.Errorf("querying child counts: %w", err)
	}

	result := make(map[string]int, len(rows))
	for _, r := range rows {
		result[r.ParentID] = r.Count
	}
	return result, nil
}
