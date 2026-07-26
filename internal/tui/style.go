package tui

import (
	"io"
	"strings"

	"github.com/charmbracelet/lipgloss"
)

type styles struct {
	accent lipgloss.Style
	green  lipgloss.Style
	amber  lipgloss.Style
	red    lipgloss.Style
	muted  lipgloss.Style
	title  lipgloss.Style
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
	}
}

func (s styles) noteCell(text string, width int) string {
	var style lipgloss.Style

	switch text {
	case "accept", "success":
		style = s.green
	case "reject", "partial", "interrupted":
		style = s.amber
	case "failed":
		style = s.red
	default:
		style = s.muted
	}

	return style.Width(width).Render(text)
}

func (s styles) blank(width int) string {
	return strings.Repeat(" ", width)
}
