package utils

import (
	"os"

	"github.com/fatih/color"
	"golang.org/x/term"
)

var (
	bold = color.New(color.Bold)
	dim  = color.New(color.Faint)

	statusColors = map[string]*color.Color{
		"ready":       color.New(color.FgGreen),
		"blocked":     color.New(color.FgYellow),
		"in_progress": color.New(color.FgCyan),
		"done":        color.New(color.Faint),
		"cancelled":   color.New(color.Faint, color.CrossedOut),
	}

	typeColors = map[string]*color.Color{
		"epic":    color.New(color.FgMagenta),
		"feature": color.New(color.FgCyan),
		"bug":     color.New(color.FgRed),
		"task":    color.New(color.Faint),
	}
)

// StatusColor returns the color associated with a task status.
func StatusColor(status string) *color.Color {
	if c, ok := statusColors[status]; ok {
		return c
	}
	return color.New(color.Reset)
}

// Bold returns text formatted as bold.
func Bold(text string) string {
	return bold.Sprint(text)
}

// Dim returns text formatted as dim/faint.
func Dim(text string) string {
	return dim.Sprint(text)
}

// TypeColor returns the color associated with a task type.
func TypeColor(taskType string) *color.Color {
	if c, ok := typeColors[taskType]; ok {
		return c
	}
	return dim
}

// TypeLabel returns a colored "[type]" label, or "" if the type is "task".
func TypeLabel(taskType string) string {
	if taskType == "task" {
		return ""
	}
	return " " + TypeColor(taskType).Sprintf("[%s]", taskType)
}

// TermWidth returns the terminal width, defaulting to 100 if detection fails.
func TermWidth() int {
	if w, _, err := term.GetSize(int(os.Stdout.Fd())); err == nil && w > 0 {
		return w
	}
	return 100
}

// Truncate shortens text to fit within maxWidth visible characters,
// appending "…" if truncated.
func Truncate(text string, maxWidth int) string {
	if maxWidth <= 1 {
		return "…"
	}
	runes := []rune(text)
	if len(runes) <= maxWidth {
		return text
	}
	return string(runes[:maxWidth-1]) + "…"
}
