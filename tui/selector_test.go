package tui

import (
	"testing"

	tea "github.com/charmbracelet/bubbletea"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestFilter_NoQuery(t *testing.T) {
	m := NewModel([]string{"alpha", "beta", "gamma"})

	assert.Len(t, m.filtered, 3)
	assert.Equal(t, "alpha", m.filtered[0].name)
	assert.Equal(t, "beta", m.filtered[1].name)
	assert.Equal(t, "gamma", m.filtered[2].name)
	for _, item := range m.filtered {
		assert.False(t, item.isCreate)
	}
}

func TestFilter_FuzzyMatch(t *testing.T) {
	m := NewModel([]string{"my-project", "my-prototype", "other-db"})
	m.textInput.SetValue("my-pro")
	m.filter()

	// Create entry should be first (default selection).
	require.True(t, len(m.filtered) >= 3)
	assert.True(t, m.filtered[0].isCreate, "create entry should be first")
	assert.Equal(t, "my-pro", m.filtered[0].name)

	// Fuzzy matches should follow.
	names := make([]string, len(m.filtered))
	for i, item := range m.filtered {
		names[i] = item.name
	}
	assert.Contains(t, names, "my-project")
	assert.Contains(t, names, "my-prototype")
}

func TestFilter_ExactMatch_NoCreate(t *testing.T) {
	m := NewModel([]string{"my-project", "other-db"})
	m.textInput.SetValue("my-project")
	m.filter()

	for _, item := range m.filtered {
		assert.False(t, item.isCreate, "exact match should not produce a create entry")
	}
}

func TestFilter_NoMatches(t *testing.T) {
	m := NewModel([]string{"alpha", "beta"})
	m.textInput.SetValue("zzz")
	m.filter()

	// Only the create entry should remain.
	require.Len(t, m.filtered, 1)
	assert.True(t, m.filtered[0].isCreate)
	assert.Equal(t, "zzz", m.filtered[0].name)
}

func TestFilter_EmptyExisting(t *testing.T) {
	m := NewModel([]string{})
	assert.Len(t, m.filtered, 0)

	m.textInput.SetValue("new-db")
	m.filter()
	require.Len(t, m.filtered, 1)
	assert.True(t, m.filtered[0].isCreate)
	assert.Equal(t, "new-db", m.filtered[0].name)
}

func sendKey(m Model, key tea.KeyType) Model {
	updated, _ := m.Update(tea.KeyMsg{Type: key})
	return updated.(Model)
}

func sendRunes(m Model, s string) Model {
	for _, r := range s {
		var updated tea.Model
		updated, _ = m.Update(tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune{r}})
		m = updated.(Model)
	}
	return m
}

func TestCursorNavigation(t *testing.T) {
	m := NewModel([]string{"a", "b", "c"})

	assert.Equal(t, 0, m.cursor)

	m = sendKey(m, tea.KeyDown)
	assert.Equal(t, 1, m.cursor)

	m = sendKey(m, tea.KeyDown)
	assert.Equal(t, 2, m.cursor)

	// Can't go past the end.
	m = sendKey(m, tea.KeyDown)
	assert.Equal(t, 2, m.cursor)

	m = sendKey(m, tea.KeyUp)
	assert.Equal(t, 1, m.cursor)

	m = sendKey(m, tea.KeyUp)
	assert.Equal(t, 0, m.cursor)

	// Can't go before the start.
	m = sendKey(m, tea.KeyUp)
	assert.Equal(t, 0, m.cursor)
}

func TestEnter_SelectsExisting(t *testing.T) {
	m := NewModel([]string{"alpha", "beta"})
	m = sendKey(m, tea.KeyDown) // cursor on "beta"
	m = sendKey(m, tea.KeyEnter)

	result, err := m.Result()
	require.NoError(t, err)
	assert.Equal(t, "beta", result.Name)
	assert.False(t, result.IsNew)
}

func TestEnter_SelectsCreateEntry(t *testing.T) {
	m := NewModel([]string{"alpha"})
	m = sendRunes(m, "new-db")

	// Create entry is first by default — just press Enter.
	assert.True(t, m.filtered[0].isCreate, "create should be the default selection")
	assert.Equal(t, 0, m.cursor)

	m = sendKey(m, tea.KeyEnter)

	result, err := m.Result()
	require.NoError(t, err)
	assert.Equal(t, "new-db", result.Name)
	assert.True(t, result.IsNew)
}

func TestEnter_DefaultIsCreate_NotMatch(t *testing.T) {
	// Even with close fuzzy matches, Enter without navigation should create new.
	m := NewModel([]string{"my-project", "my-prototype"})
	m = sendRunes(m, "my-pro")

	// Don't navigate — just press Enter immediately.
	m = sendKey(m, tea.KeyEnter)

	result, err := m.Result()
	require.NoError(t, err)
	assert.Equal(t, "my-pro", result.Name)
	assert.True(t, result.IsNew, "default Enter should create, not select a match")
}

func TestEnter_NavigateToMatch(t *testing.T) {
	// User must arrow-down to select an existing match.
	m := NewModel([]string{"my-project", "my-prototype"})
	m = sendRunes(m, "my-pro")

	m = sendKey(m, tea.KeyDown) // move past create entry to first match
	m = sendKey(m, tea.KeyEnter)

	result, err := m.Result()
	require.NoError(t, err)
	assert.False(t, result.IsNew, "navigating to a match should select existing")
}

func TestEnter_EmptyInput_NoItems(t *testing.T) {
	m := NewModel([]string{})
	m = sendKey(m, tea.KeyEnter)

	// Should not quit — empty input with nothing to select.
	assert.False(t, m.quitting)
}

func TestEsc_Cancels(t *testing.T) {
	m := NewModel([]string{"alpha"})
	m = sendKey(m, tea.KeyEsc)

	result, err := m.Result()
	assert.Error(t, err)
	assert.Nil(t, result)
	assert.True(t, m.quitting)
}

func TestView_ContainsItems(t *testing.T) {
	m := NewModel([]string{"alpha", "beta"})
	view := m.View()

	assert.Contains(t, view, "alpha")
	assert.Contains(t, view, "beta")
	assert.Contains(t, view, "Find or create")
}

func TestView_EmptyExisting(t *testing.T) {
	m := NewModel([]string{})
	view := m.View()

	assert.Contains(t, view, "Database name:")
}

func TestHighlightMatches(t *testing.T) {
	result := highlightMatches("my-project", []int{0, 1, 2})
	// Should contain the full name.
	assert.Contains(t, result, "my")
	assert.Contains(t, result, "project")
}

func TestHighlightMatches_NoPositions(t *testing.T) {
	result := highlightMatches("alpha", nil)
	assert.Equal(t, "alpha", result)
}

func TestCursorResets_OnFilter(t *testing.T) {
	m := NewModel([]string{"alpha", "beta", "gamma"})
	m = sendKey(m, tea.KeyDown)
	m = sendKey(m, tea.KeyDown)
	assert.Equal(t, 2, m.cursor)

	// Typing should reset cursor to 0.
	m = sendRunes(m, "a")
	assert.Equal(t, 0, m.cursor)
}
