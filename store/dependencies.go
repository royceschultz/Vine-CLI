package store

import "fmt"

type Dependency struct {
	TaskID      string `db:"task_id"       json:"task_id"`
	DependsOnID string `db:"depends_on_id" json:"depends_on_id"`
	CreatedAt   string `db:"created_at"    json:"created_at"`
}

func (s *Store) AddDependency(taskID, dependsOnID string) error {
	_, err := s.db.Exec(
		"INSERT INTO dependencies (task_id, depends_on_id) VALUES (?, ?)",
		taskID, dependsOnID,
	)
	if err != nil {
		return fmt.Errorf("adding dependency %s -> %s: %w", taskID, dependsOnID, err)
	}
	return nil
}

func (s *Store) RemoveDependency(taskID, dependsOnID string) error {
	result, err := s.db.Exec(
		"DELETE FROM dependencies WHERE task_id = ? AND depends_on_id = ?",
		taskID, dependsOnID,
	)
	if err != nil {
		return fmt.Errorf("removing dependency %s -> %s: %w", taskID, dependsOnID, err)
	}
	n, _ := result.RowsAffected()
	if n == 0 {
		return fmt.Errorf("dependency %s -> %s not found", taskID, dependsOnID)
	}
	return nil
}

func (s *Store) DependenciesOf(taskID string) ([]Dependency, error) {
	var deps []Dependency
	err := s.db.Select(&deps,
		"SELECT * FROM dependencies WHERE task_id = ? ORDER BY depends_on_id", taskID)
	if err != nil {
		return nil, fmt.Errorf("listing dependencies for task %s: %w", taskID, err)
	}
	return deps, nil
}

// DependentsOf returns dependencies where other tasks depend on the given task.
func (s *Store) DependentsOf(taskID string) ([]Dependency, error) {
	var deps []Dependency
	err := s.db.Select(&deps,
		"SELECT * FROM dependencies WHERE depends_on_id = ? ORDER BY task_id", taskID)
	if err != nil {
		return nil, fmt.Errorf("listing dependents of task %s: %w", taskID, err)
	}
	return deps, nil
}
