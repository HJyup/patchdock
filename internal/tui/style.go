package tui

import (
	"io"

	"github.com/charmbracelet/lipgloss"
)

type styles struct {
	accent lipgloss.Style
	green  lipgloss.Style
	amber  lipgloss.Style
	red    lipgloss.Style
	muted  lipgloss.Style
	title  lipgloss.Style
	strong lipgloss.Style
}

func newStyles(out io.Writer) styles {
	renderer := lipgloss.NewRenderer(out)
	colour := func(code string) lipgloss.Style {
		return renderer.NewStyle().Foreground(lipgloss.Color(code))
	}

	return styles{
		accent: colour("39"), // patchdock blue
		green:  colour("42"), // accepted, succeeded
		amber:  colour("214"),
		red:    colour("203"),
		muted:  colour("245"), // elapsed times, activity, paths
		title:  colour("39").Bold(true),
		strong: renderer.NewStyle().Bold(true),
	}
}
