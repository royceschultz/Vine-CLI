package utils

import "github.com/fatih/color"

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
