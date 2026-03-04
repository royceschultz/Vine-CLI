package utils

import (
	"testing"

	"github.com/stretchr/testify/assert"
)

func TestGenerateID_Length(t *testing.T) {
	id := GenerateID(IDLength)
	assert.Len(t, id, IDLength)
}

func TestGenerateID_Characters(t *testing.T) {
	for i := 0; i < 100; i++ {
		id := GenerateID(IDLength)
		for _, c := range id {
			assert.Contains(t, idChars, string(c))
		}
	}
}

func TestGenerateID_Unique(t *testing.T) {
	// With 4 chars (36^4 = ~1.6M possibilities), collisions become likely
	// around ~1000 IDs (birthday paradox). 100 is a safe test size.
	seen := make(map[string]bool)
	for i := 0; i < 100; i++ {
		id := GenerateID(IDLength)
		assert.False(t, seen[id], "collision on %s after %d generations", id, i)
		seen[id] = true
	}
}

func TestFormatTaskID(t *testing.T) {
	assert.Equal(t, "myproject-k7x2", FormatTaskID("myproject", "k7x2"))
}

func TestParseTaskID_WithProject(t *testing.T) {
	project, id := ParseTaskID("myproject-k7x2")
	assert.Equal(t, "myproject", project)
	assert.Equal(t, "k7x2", id)
}

func TestParseTaskID_BareID(t *testing.T) {
	project, id := ParseTaskID("k7x2")
	assert.Equal(t, "", project)
	assert.Equal(t, "k7x2", id)
}

func TestParseTaskID_MultiDash(t *testing.T) {
	project, id := ParseTaskID("my-cool-project-k7x2")
	assert.Equal(t, "my-cool-project", project)
	assert.Equal(t, "k7x2", id)
}
