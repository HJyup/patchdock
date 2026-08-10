package tui

import (
	"fmt"
	"strings"
	"time"

	"github.com/charmbracelet/bubbles/spinner"
	tea "github.com/charmbracelet/bubbletea"
	"github.com/charmbracelet/x/ansi"
)

type (
	headerMsg struct{ info RunInfo }
	startMsg  struct{ label string }
	detailMsg struct{ activity string }
	noteMsg   struct{ text string }
	finishMsg struct{ mark string }
)

// minDetail keeps the activity line legible on an implausibly narrow terminal
const minDetail = 20

type model struct {
	styles  styles
	spin    spinner.Model
	timeout time.Duration
	info    RunInfo
	label   string
	summary string
	detail  string
	started time.Time
	active  bool
	width   int
}

func newModel(s styles, timeout time.Duration) model {
	spin := spinner.New()
	// MiniDot rather than Dot: Dot's frames carry a trailing space, which would
	// double up against the separator below
	spin.Spinner = spinner.MiniDot
	spin.Style = s.accent

	return model{styles: s, spin: spin, timeout: timeout, width: fallbackCols}
}

func (m model) Init() tea.Cmd {
	return m.spin.Tick
}

func (m model) Update(msg tea.Msg) (tea.Model, tea.Cmd) {
	switch msg := msg.(type) {
	case tea.WindowSizeMsg:
		// A pty that cannot report its size sends 0, which would shrink every
		// truncation to nothing. Keep the fallback instead
		if msg.Width > 0 {
			m.width = msg.Width
		}
		return m, nil

	case headerMsg:
		m.info = msg.info
		return m, nil

	case startMsg:
		m.label, m.detail, m.started, m.active = msg.label, "", time.Now(), true
		return m, nil

	case detailMsg:
		m.detail = msg.activity
		return m, nil

	// a note replaces its predecessor: the plan is the run's account of itself
	// until a reviewer supersedes it with a verdict
	case noteMsg:
		m.summary = msg.text
		return m, nil

	case finishMsg:
		m.active, m.detail = false, ""
		return m, nil

	case spinner.TickMsg:
		var cmd tea.Cmd
		m.spin, cmd = m.spin.Update(msg)
		return m, cmd
	}

	return m, nil
}

func (m model) View() string {
	var b strings.Builder

	if header := headerLines(m.info, m.styles, m.width); len(header) > 0 {
		fmt.Fprintf(&b, "%s\n\n", strings.Join(header, "\n"))
	}
	if !m.active {
		return b.String()
	}

	elapsed := time.Since(m.started)
	fmt.Fprintf(&b, "%s%s %s  %s  %s",
		gutter,
		m.spin.View(),
		m.label,
		m.styles.muted.Render(pad(short(elapsed), timeWidth)),
		remaining(m.styles, m.timeout, elapsed))

	for _, line := range []string{m.child(m.summary), m.child(m.detail)} {
		if line == "" {
			continue
		}
		fmt.Fprintf(&b, "\n%s%s", childIndent, m.styles.muted.Render(line))
	}

	return b.String()
}

func (m model) child(text string) string {
	return ansi.Truncate(oneLine(text), max(m.width-len(childIndent)-1, minDetail), "…")
}
