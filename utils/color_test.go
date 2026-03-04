package utils

import (
	"testing"

	"github.com/stretchr/testify/assert"
)

func TestStatusColor_KnownStatuses(t *testing.T) {
	known := []string{"ready", "blocked", "in_progress", "done", "cancelled"}
	for _, status := range known {
		c := StatusColor(status)
		assert.NotNil(t, c, "StatusColor(%q) should not be nil", status)
	}
}

func TestStatusColor_Unknown(t *testing.T) {
	c := StatusColor("unknown_status")
	assert.NotNil(t, c)
}

func TestStatusColor_DifferentColors(t *testing.T) {
	// ready and blocked should use different colors.
	ready := StatusColor("ready")
	blocked := StatusColor("blocked")
	assert.NotEqual(t, ready, blocked)
}

func TestBold(t *testing.T) {
	result := Bold("hello")
	assert.Contains(t, result, "hello")
}

func TestDim(t *testing.T) {
	result := Dim("hello")
	assert.Contains(t, result, "hello")
}
