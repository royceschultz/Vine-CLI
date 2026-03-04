package tui

import (
	"bufio"
	"fmt"
	"os"
	"strconv"
	"strings"

	"github.com/charmbracelet/bubbles/textinput"
	tea "github.com/charmbracelet/bubbletea"
	"github.com/charmbracelet/lipgloss"
	"github.com/mattn/go-isatty"
	"github.com/sahilm/fuzzy"
)

// SelectorResult holds the outcome of the selector interaction.
type SelectorResult struct {
	Name  string // The chosen or typed database name.
	IsNew bool   // True if the user is creating a new database.
}

// item represents a visible entry in the filtered list.
type item struct {
	name     string
	isCreate bool  // true for the synthetic "+ Create ..." entry
	matches  []int // character indices to highlight (from fuzzy match)
}

const maxVisible = 10

var (
	cursorStyle  = lipgloss.NewStyle().Foreground(lipgloss.Color("10")) // green
	matchStyle   = lipgloss.NewStyle().Bold(true)
	dimStyle     = lipgloss.NewStyle().Faint(true)
	createStyle  = lipgloss.NewStyle().Foreground(lipgloss.Color("10")) // green
	selectedStyle = lipgloss.NewStyle().Foreground(lipgloss.Color("10")).Bold(true)
)

// Model is the bubbletea model for the fuzzy-search selector.
type Model struct {
	textInput  textinput.Model
	allItems   []string // all existing database names
	filtered   []item   // currently visible items after filtering
	cursor     int      // index into filtered
	offset     int      // scroll offset for viewport
	chosen     *SelectorResult
	err        error
	quitting   bool
}

// NewModel creates a selector with the given existing items.
func NewModel(existing []string) Model {
	ti := textinput.New()
	ti.Focus()
	ti.CharLimit = 128
	if len(existing) > 0 {
		ti.Placeholder = "type to filter..."
	}

	m := Model{
		textInput: ti,
		allItems:  existing,
	}
	m.filter()
	return m
}

// Init satisfies tea.Model.
func (m Model) Init() tea.Cmd {
	return textinput.Blink
}

// Update satisfies tea.Model.
func (m Model) Update(msg tea.Msg) (tea.Model, tea.Cmd) {
	switch msg := msg.(type) {
	case tea.KeyMsg:
		switch msg.Type {
		case tea.KeyCtrlC, tea.KeyEsc:
			m.err = fmt.Errorf("cancelled")
			m.quitting = true
			return m, tea.Quit

		case tea.KeyEnter:
			if len(m.filtered) == 0 {
				// Nothing to select and no text — ignore.
				if m.textInput.Value() == "" {
					return m, nil
				}
				// Text typed but no matches — create new.
				m.chosen = &SelectorResult{
					Name:  m.textInput.Value(),
					IsNew: true,
				}
				m.quitting = true
				return m, tea.Quit
			}
			selected := m.filtered[m.cursor]
			m.chosen = &SelectorResult{
				Name:  selected.name,
				IsNew: selected.isCreate,
			}
			m.quitting = true
			return m, tea.Quit

		case tea.KeyUp:
			if m.cursor > 0 {
				m.cursor--
				if m.cursor < m.offset {
					m.offset = m.cursor
				}
			}
			return m, nil

		case tea.KeyDown:
			if m.cursor < len(m.filtered)-1 {
				m.cursor++
				if m.cursor >= m.offset+maxVisible {
					m.offset = m.cursor - maxVisible + 1
				}
			}
			return m, nil
		}
	}

	// Delegate to text input.
	prevValue := m.textInput.Value()
	var cmd tea.Cmd
	m.textInput, cmd = m.textInput.Update(msg)

	// Re-filter if the text changed.
	if m.textInput.Value() != prevValue {
		m.filter()
		m.cursor = 0
		m.offset = 0
	}

	return m, cmd
}

// View satisfies tea.Model.
func (m Model) View() string {
	if m.quitting {
		return ""
	}

	var b strings.Builder

	// Header and input.
	if len(m.allItems) > 0 {
		b.WriteString("  Find or create a database:\n")
	} else {
		b.WriteString("  Database name:\n")
	}
	b.WriteString("  > " + m.textInput.View() + "\n")

	// Filtered list.
	if len(m.filtered) == 0 && len(m.allItems) > 0 {
		b.WriteString("\n")
		b.WriteString(dimStyle.Render("    no matches") + "\n")
	} else {
		b.WriteString("\n")
		end := m.offset + maxVisible
		if end > len(m.filtered) {
			end = len(m.filtered)
		}
		for i := m.offset; i < end; i++ {
			entry := m.filtered[i]
			cursor := "  "
			if i == m.cursor {
				cursor = cursorStyle.Render("> ")
			}

			var line string
			if entry.isCreate {
				nameStr := fmt.Sprintf("+ Create %q", entry.name)
				if i == m.cursor {
					line = selectedStyle.Render(nameStr) + " " + dimStyle.Render("(new)")
				} else {
					line = createStyle.Render(nameStr) + " " + dimStyle.Render("(new)")
				}
			} else {
				nameStr := highlightMatches(entry.name, entry.matches)
				label := dimStyle.Render("(existing)")
				line = nameStr + " " + label
			}

			b.WriteString("  " + cursor + line + "\n")
		}

		// Scroll indicators.
		if m.offset > 0 {
			b.WriteString(dimStyle.Render("    ↑ more") + "\n")
		}
		if end < len(m.filtered) {
			b.WriteString(dimStyle.Render("    ↓ more") + "\n")
		}
	}

	// Help line.
	b.WriteString("\n")
	if len(m.allItems) > 0 {
		b.WriteString(dimStyle.Render("  ↑/↓ navigate · enter confirm · esc cancel") + "\n")
	} else {
		b.WriteString(dimStyle.Render("  enter confirm · esc cancel") + "\n")
	}

	return b.String()
}

// Result returns the selection outcome after the program exits.
func (m Model) Result() (*SelectorResult, error) {
	if m.err != nil {
		return nil, m.err
	}
	return m.chosen, nil
}

// filter applies fuzzy matching and rebuilds the filtered list.
func (m *Model) filter() {
	query := m.textInput.Value()

	m.filtered = nil

	if query == "" {
		// Show all items unfiltered.
		for _, name := range m.allItems {
			m.filtered = append(m.filtered, item{name: name})
		}
		return
	}

	// Fuzzy match against existing items.
	matches := fuzzy.Find(query, m.allItems)
	exactMatch := false
	for _, match := range matches {
		if strings.EqualFold(match.Str, query) {
			exactMatch = true
		}
	}

	// Put "create new" first so it's the default selection.
	if !exactMatch {
		m.filtered = append(m.filtered, item{
			name:     query,
			isCreate: true,
		})
	}

	for _, match := range matches {
		m.filtered = append(m.filtered, item{
			name:    match.Str,
			matches: match.MatchedIndexes,
		})
	}
}

// highlightMatches renders a name with matched characters in bold.
func highlightMatches(name string, positions []int) string {
	if len(positions) == 0 {
		return name
	}

	posSet := make(map[int]bool, len(positions))
	for _, p := range positions {
		posSet[p] = true
	}

	var b strings.Builder
	var run strings.Builder
	inMatch := false

	for i, ch := range name {
		isMatch := posSet[i]
		if isMatch != inMatch {
			// Flush the current run.
			if inMatch {
				b.WriteString(matchStyle.Render(run.String()))
			} else {
				b.WriteString(run.String())
			}
			run.Reset()
			inMatch = isMatch
		}
		run.WriteRune(ch)
	}
	// Flush final run.
	if inMatch {
		b.WriteString(matchStyle.Render(run.String()))
	} else {
		b.WriteString(run.String())
	}

	return b.String()
}

// SelectDatabase runs the interactive fuzzy selector and returns the chosen name.
// If stdin is not a TTY, it falls back to a plain numbered-list prompt.
func SelectDatabase(existing []string) (*SelectorResult, error) {
	if !isatty.IsTerminal(os.Stdin.Fd()) && !isatty.IsCygwinTerminal(os.Stdin.Fd()) {
		return selectDatabasePlain(existing)
	}

	m := NewModel(existing)
	p := tea.NewProgram(m)
	finalModel, err := p.Run()
	if err != nil {
		return nil, fmt.Errorf("selector: %w", err)
	}
	return finalModel.(Model).Result()
}

// selectDatabasePlain is the non-TTY fallback using a numbered list.
func selectDatabasePlain(existing []string) (*SelectorResult, error) {
	reader := bufio.NewReader(os.Stdin)

	if len(existing) > 0 {
		fmt.Println()
		fmt.Println("Existing databases:")
		for i, name := range existing {
			fmt.Printf("  [%d] %s\n", i+1, name)
		}
		fmt.Println()
		fmt.Print("Enter a number to join, or type a new name: ")
	} else {
		fmt.Print("Database name: ")
	}

	input, _ := reader.ReadString('\n')
	input = strings.TrimSpace(input)
	if input == "" {
		return nil, fmt.Errorf("database name cannot be empty")
	}

	// Check if the input is a number referencing an existing database.
	if n, err := strconv.Atoi(input); err == nil {
		if n >= 1 && n <= len(existing) {
			return &SelectorResult{Name: existing[n-1], IsNew: false}, nil
		}
		return nil, fmt.Errorf("invalid selection: %s (expected 1-%d)", input, len(existing))
	}

	// Check if the typed name matches an existing database.
	isNew := true
	for _, name := range existing {
		if strings.EqualFold(name, input) {
			isNew = false
			break
		}
	}

	return &SelectorResult{Name: input, IsNew: isNew}, nil
}
