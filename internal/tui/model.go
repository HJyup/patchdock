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
	finishMsg struct{ note, mark string }
)

// minDetail keeps the activity line legible on an implausibly narrow terminal
const minDetail = 20

// model holds the finished steps rather than handing them to tea.Println.
// Println commits asynchronously, so the last line of a run races Quit and is
// lost; keeping the lines in the view makes the final frame authoritative. The
// cost is that the live region grows with the run, which is bounded here by the
// pipeline itself: one planner plus two stages per attempt
type model struct {
	styles  styles
	spin    spinner.Model
	timeout time.Duration
	info    RunInfo
	done    []string
	label   string
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

	case noteMsg:
		m.done = append(m.done, childIndent+m.styles.muted.Render(msg.text))
		return m, nil

	case finishMsg:
		if !m.active {
			return m, nil
		}
		m.done = append(m.done, m.commit(msg.mark, msg.note))
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
	for _, line := range m.done {
		fmt.Fprintf(&b, "%s\n", line)
	}

	if !m.active {
		return b.String()
	}

	elapsed := time.Since(m.started)
	fmt.Fprintf(&b, "%s%s %s %s  %s  %s",
		gutter,
		m.spin.View(),
		m.label,
		m.styles.blank(noteWidth),
		m.styles.muted.Render(pad(short(elapsed), timeWidth)),
		remaining(m.styles, m.timeout, elapsed))

	if m.detail != "" {
		// Truncation is display-width aware, so a wide rune cannot overflow the
		// row and force the renderer to reason about a wrap it did not plan
		activity := ansi.Truncate(m.detail, max(m.width-len(childIndent)-1, minDetail), "…")
		fmt.Fprintf(&b, "\n%s%s", childIndent, m.styles.muted.Render(activity))
	}

	return b.String()
}

func (m model) commit(mark, note string) string {
	return fmt.Sprintf("%s%s %s %s  %s",
		gutter,
		mark,
		m.label,
		m.styles.noteCell(note, noteWidth),
		m.styles.muted.Render(short(time.Since(m.started))))
}
