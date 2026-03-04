package store

import (
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func newTestStore(t *testing.T) *Store {
	t.Helper()
	s, err := OpenMemory()
	require.NoError(t, err)
	t.Cleanup(func() { s.Close() })
	return s
}

func TestOpenMemory(t *testing.T) {
	s := newTestStore(t)
	assert.NotNil(t, s)
}

func TestMigrationsApplied(t *testing.T) {
	s := newTestStore(t)

	var count int
	err := s.db.Get(&count, "SELECT COUNT(*) FROM schema_migrations")
	require.NoError(t, err)
	assert.GreaterOrEqual(t, count, 1)
}

func TestMigrationsIdempotent(t *testing.T) {
	s := newTestStore(t)

	// Running migrate again should not error.
	err := s.migrate()
	assert.NoError(t, err)
}
