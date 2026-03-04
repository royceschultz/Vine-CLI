package store

import "fmt"

type StatusCount struct {
	Status string `db:"effective_status" json:"status"`
	Count  int    `db:"count"            json:"count"`
}

type StatusTypeCount struct {
	Status string `db:"effective_status" json:"status"`
	Type   string `db:"type"             json:"type"`
	Count  int    `db:"count"            json:"count"`
}

const effectiveStatusExpr = `
	CASE
		WHEN t.status = 'open' AND EXISTS (
			SELECT 1 FROM dependencies d
			JOIN tasks dep ON dep.id = d.depends_on_id
			WHERE d.task_id = t.id AND dep.status NOT IN ('done', 'cancelled')
		) THEN 'blocked'
		WHEN t.status = 'open' THEN 'ready'
		ELSE t.status
	END
`

// TaskSummary returns task counts grouped by effective status.
func (s *Store) TaskSummary() ([]StatusCount, error) {
	var counts []StatusCount
	query := fmt.Sprintf(`
		SELECT %s AS effective_status, COUNT(*) AS count
		FROM tasks t
		GROUP BY effective_status
		ORDER BY
			CASE effective_status
				WHEN 'ready' THEN 1
				WHEN 'blocked' THEN 2
				WHEN 'in_progress' THEN 3
				WHEN 'done' THEN 4
				WHEN 'cancelled' THEN 5
			END
	`, effectiveStatusExpr)
	if err := s.db.Select(&counts, query); err != nil {
		return nil, fmt.Errorf("querying task summary: %w", err)
	}
	return counts, nil
}

// TaskSummaryDetailed returns task counts grouped by effective status and type.
func (s *Store) TaskSummaryDetailed() ([]StatusTypeCount, error) {
	var counts []StatusTypeCount
	query := fmt.Sprintf(`
		SELECT %s AS effective_status, t.type, COUNT(*) AS count
		FROM tasks t
		GROUP BY effective_status, t.type
		ORDER BY
			CASE effective_status
				WHEN 'ready' THEN 1
				WHEN 'blocked' THEN 2
				WHEN 'in_progress' THEN 3
				WHEN 'done' THEN 4
				WHEN 'cancelled' THEN 5
			END,
			t.type
	`, effectiveStatusExpr)
	if err := s.db.Select(&counts, query); err != nil {
		return nil, fmt.Errorf("querying detailed task summary: %w", err)
	}
	return counts, nil
}
