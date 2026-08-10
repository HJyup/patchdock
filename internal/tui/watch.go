package tui

import (
	"context"
	"errors"
	"fmt"
	"io"
	"maps"
	"slices"
	"strings"
	"time"

	"github.com/charmbracelet/bubbles/spinner"
	tea "github.com/charmbracelet/bubbletea"
	"github.com/charmbracelet/x/ansi"

	"github.com/HJyup/patchdock/internal/daemon/api"
)

// titleMax caps the task column so one long prompt cannot push every status
// off screen
const titleMax = 44

// StreamFunc feeds the dashboard. It should block, calling onSnapshot for
// every snapshot, until ctx is cancelled or the stream breaks
type StreamFunc func(ctx context.Context, onSnapshot func(api.Snapshot) error) error

// Watch runs the dashboard until the user quits or the stream fails. Every
// snapshot replaces the whole view: the daemon publishes full state, so there
// is nothing to accumulate client-side
func Watch(ctx context.Context, in io.Reader, out io.Writer, stream StreamFunc) error {
	if !Interactive(in, out) {
		return errors.New("dock watch needs a terminal")
	}

	ctx, cancel := context.WithCancel(ctx)
	defer cancel()

	// The dashboard owns the whole terminal while it is open, like any
	// long-lived monitor, and hands the scrollback base state on quit
	program := tea.NewProgram(
		newWatchModel(newStyles(out)),
		tea.WithInput(in),
		tea.WithOutput(out),
		tea.WithAltScreen(),
	)

	// The stream owns one channel send; the program is told to quit on
	// failure so Run below unblocks and the error surfaces after cleanup
	streamErr := make(chan error, 1)
	go func() {
		err := stream(ctx, func(snap api.Snapshot) error {
			program.Send(snapshotMsg{snap: snap})
			return nil
		})

		if ctx.Err() != nil { // the user quit; whatever the stream says is noise
			streamErr <- nil
			return
		}
		if err == nil {
			err = errors.New("daemon closed the stream")
		}

		streamErr <- err
		program.Send(streamFailedMsg{})
	}()

	_, runErr := program.Run()
	cancel()

	if err := <-streamErr; err != nil {
		return fmt.Errorf("watch runs: %w", err)
	}
	if runErr != nil && !errors.Is(runErr, tea.ErrInterrupted) && !errors.Is(runErr, tea.ErrProgramKilled) {
		return runErr
	}
	return nil
}

type (
	snapshotMsg     struct{ snap api.Snapshot }
	streamFailedMsg struct{}
)

type watchModel struct {
	styles styles
	spin   spinner.Model
	runs   []api.Run
	seen   bool // one snapshot has arrived; an empty list now means "no runs"
	width  int
	height int
}

func newWatchModel(s styles) watchModel {
	spin := spinner.New()
	spin.Spinner = spinner.MiniDot
	spin.Style = s.accent

	return watchModel{styles: s, spin: spin, width: fallbackCols}
}

func (m watchModel) Init() tea.Cmd {
	return m.spin.Tick
}

func (m watchModel) Update(msg tea.Msg) (tea.Model, tea.Cmd) {
	switch msg := msg.(type) {
	case tea.WindowSizeMsg:
		if msg.Width > 0 {
			m.width = msg.Width
		}
		if msg.Height > 0 {
			m.height = msg.Height
		}
		return m, nil

	case tea.KeyMsg:
		switch msg.String() {
		case "q", "esc", "ctrl+c":
			return m, tea.Quit
		}
		return m, nil

	case snapshotMsg:
		m.runs = msg.snap.Runs
		m.seen = true
		return m, nil

	case streamFailedMsg:
		return m, tea.Quit

	// the spinner tick doubles as the clock: every frame repaints, so the
	// elapsed columns advance without a timer of their own
	case spinner.TickMsg:
		var cmd tea.Cmd
		m.spin, cmd = m.spin.Update(msg)
		return m, cmd
	}

	return m, nil
}

func (m watchModel) View() string {
	var b strings.Builder

	fmt.Fprintf(&b, "\n%s\n", m.header())

	switch {
	case !m.seen:
		fmt.Fprintf(&b, "\n%s%s\n", gutter, m.styles.muted.Render("connecting…"))

	case len(m.runs) == 0:
		fmt.Fprintf(&b, "\n%s%s\n", gutter,
			m.styles.muted.Render(`no runs — queue one with dock run -p "…"`))

	default:
		titleWidth := m.titleWidth()
		for _, group := range groupRuns(m.runs) {
			fmt.Fprintf(&b, "\n%s%s\n", gutter, m.styles.strong.Render(group.name))

			for _, run := range group.runs {
				b.WriteString(m.runLine(run, titleWidth))
				if child := m.childLine(run); child != "" {
					fmt.Fprintf(&b, "%s%s\n", childIndent, child)
				}
			}
		}
	}

	return m.pinFooter(b.String())
}

// header sets the wordmark against the tally, one on each margin, the way a
// status bar reads: identity left, numbers right
func (m watchModel) header() string {
	left := gutter + m.styles.title.Render("patchdock")
	tally := m.styles.muted.Render(m.tally())

	gap := m.width - ansi.StringWidth(left) - ansi.StringWidth(tally) - len(gutter)
	if gap < 2 {
		return left + "  " + tally
	}
	return left + strings.Repeat(" ", gap) + tally
}

// pinFooter holds the quit hint on the bottom row of the screen, however
// short the run list is
func (m watchModel) pinFooter(body string) string {
	footer := gutter + m.styles.muted.Render("q to quit")

	if fill := m.height - strings.Count(body, "\n") - 2; fill > 0 {
		body += strings.Repeat("\n", fill)
	} else {
		body += "\n"
	}

	return body + footer
}

func (m watchModel) runLine(run api.Run, titleWidth int) string {
	title := ansi.Truncate(oneLine(run.Title), titleWidth, "…")

	line := fmt.Sprintf("%s%s %s  %s  %s",
		subIndent,
		m.mark(run.Status),
		pad(title, titleWidth),
		m.statusWord(run.Status),
		m.styles.muted.Render(pad(short(elapsed(run, time.Now())), timeWidth)))

	if run.Attempt > 1 {
		line += m.styles.muted.Render(fmt.Sprintf(" (attempt %d)", run.Attempt))
	}

	return line + "\n"
}

// childLine picks the one line of context a run deserves under its row: live
// activity while it works, the patch it published once it has, and the
// summary — plan or failure — otherwise. Free text is flattened and truncated
// before any styling, so escape sequences never reach oneLine's control-rune
// filter
func (m watchModel) childLine(run api.Run) string {
	if run.Status == api.StatusSucceeded {
		if stat := m.patchStat(run); stat != "" {
			return stat
		}
	}

	text := run.Summary
	if !api.IsFinilised(run.Status) && run.Activity != "" {
		text = run.Activity
	}

	if text = oneLine(text); text == "" {
		return ""
	}
	return m.styles.muted.Render(ansi.Truncate(text, max(m.width-len(childIndent)-1, minDetail), "…"))
}

// patchStat is assembled from bounded parts rather than free text, so it is
// returned styled and skips the flattening above
func (m watchModel) patchStat(run api.Run) string {
	if run.Patch == nil || run.Patch.Files == 0 {
		return ""
	}

	stat := plural(run.Patch.Files, "file")
	if run.Patch.Additions > 0 {
		stat += "  " + m.styles.green.Render(fmt.Sprintf("+%d", run.Patch.Additions))
	}
	if run.Patch.Deletions > 0 {
		stat += "  " + m.styles.red.Render(fmt.Sprintf("-%d", run.Patch.Deletions))
	}
	if run.Branch != "" {
		stat += "  " + arrowSign + "  " + run.Branch
	}

	return stat
}

func (m watchModel) mark(status api.Status) string {
	switch status {
	case api.StatusQueued:
		return m.styles.muted.Render(queuedSign)
	case api.StatusSucceeded:
		return m.styles.green.Render(sucessSign)
	case api.StatusFailed, api.StatusRejected:
		return m.styles.red.Render(rejectSign)
	case api.StatusCancelled:
		return m.styles.amber.Render(rejectSign)
	default:
		return m.spin.View()
	}
}

// statusWidth fits the longest status name, so the elapsed column lines up
// across rows
const statusWidth = len(api.StatusPublishing)

// statusWord keeps colour for outcomes only. A working run already announces
// itself through the spinner; painting its status too made every busy row
// compete with the wordmark
func (m watchModel) statusWord(status api.Status) string {
	word := pad(string(status), statusWidth)

	switch status {
	case api.StatusQueued:
		return m.styles.muted.Render(word)
	case api.StatusSucceeded:
		return m.styles.green.Render(word)
	case api.StatusFailed, api.StatusRejected:
		return m.styles.red.Render(word)
	case api.StatusCancelled:
		return m.styles.amber.Render(word)
	default:
		return word
	}
}

func (m watchModel) tally() string {
	var queued, running, done int
	for _, run := range m.runs {
		switch {
		case run.Status == api.StatusQueued:
			queued++
		case api.IsFinilised(run.Status):
			done++
		default:
			running++
		}
	}

	parts := make([]string, 0, 3)
	if running > 0 {
		parts = append(parts, fmt.Sprintf("%d running", running))
	}
	if queued > 0 {
		parts = append(parts, fmt.Sprintf("%d queued", queued))
	}
	if done > 0 {
		parts = append(parts, fmt.Sprintf("%d done", done))
	}

	if len(parts) == 0 {
		return "idle"
	}
	return strings.Join(parts, " · ")
}

// titleWidth sizes the task column to its content, within what the terminal
// and titleMax leave for it
func (m watchModel) titleWidth() int {
	widest := 0
	for _, run := range m.runs {
		widest = max(widest, ansi.StringWidth(oneLine(run.Title)))
	}

	// subIndent, the mark, the status and elapsed columns, and their gaps
	overhead := len(subIndent) + 2 + 2 + statusWidth + 2 + timeWidth
	return max(min(widest, titleMax, m.width-overhead), minDetail)
}

// elapsed reports the time a run has spent on the phase it is in: waiting if
// queued, working if started, and its total working time once finished
func elapsed(run api.Run, now time.Time) time.Duration {
	switch {
	case run.FinishedAt != nil && run.StartedAt != nil:
		return run.FinishedAt.Sub(*run.StartedAt)
	case run.FinishedAt != nil:
		return run.FinishedAt.Sub(run.QueuedAt)
	case run.StartedAt != nil:
		return now.Sub(*run.StartedAt)
	default:
		return now.Sub(run.QueuedAt)
	}
}

type repoGroup struct {
	name string
	runs []api.Run
}

// groupRuns buckets runs by repo, repos alphabetically and each repo's runs
// oldest first, so rows keep stable positions as snapshots replace each
// other. Only the repo's base name is shown: the full path said little at the
// cost of a line's worth of noise per group
func groupRuns(runs []api.Run) []repoGroup {
	byRepo := make(map[string][]api.Run)
	for _, run := range runs {
		byRepo[run.Repo] = append(byRepo[run.Repo], run)
	}

	groups := make([]repoGroup, 0, len(byRepo))
	for _, path := range slices.Sorted(maps.Keys(byRepo)) {
		bucket := byRepo[path]
		slices.SortFunc(bucket, func(a, b api.Run) int {
			if c := a.QueuedAt.Compare(b.QueuedAt); c != 0 {
				return c
			}
			return strings.Compare(a.ID, b.ID)
		})

		groups = append(groups, repoGroup{
			name: path[strings.LastIndexByte(path, '/')+1:],
			runs: bucket,
		})
	}

	return groups
}
