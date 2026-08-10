package tui

import (
	"fmt"
	"maps"
	"slices"
	"strings"
	"time"

	"github.com/charmbracelet/bubbles/spinner"
	tea "github.com/charmbracelet/bubbletea"
	"github.com/charmbracelet/x/ansi"

	"github.com/HJyup/patchdock/internal/daemon/api"
)

type watchModel struct {
	styles styles
	spin   spinner.Model
	runs   []api.Run
	seen   bool // one snapshot has arrived; an empty list now means "no runs"
	footer string
	width  int
	height int
}

func newWatchModel(s styles, footer string) watchModel {
	spin := spinner.New()
	spin.Spinner = spinner.MiniDot
	spin.Style = s.accent

	return watchModel{styles: s, spin: spin, footer: footer, width: fallbackCols}
}

func (m watchModel) update(msg tea.Msg) (watchModel, tea.Cmd) {
	switch msg := msg.(type) {
	case tea.WindowSizeMsg:
		if msg.Width > 0 {
			m.width = msg.Width
		}
		if msg.Height > 0 {
			m.height = msg.Height
		}
		return m, nil

	case snapshotMsg:
		m.runs = msg.snap.Runs
		m.seen = true
		return m, nil

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
		for _, group := range groupRuns(m.runs) {
			fmt.Fprintf(&b, "\n%s%s\n", gutter, m.styles.strong.Render(group.name))

			// every run gets a blank line above it: rows carry two lines of
			// their own, so without the gap neighbouring runs read as one block
			for _, run := range group.runs {
				b.WriteString("\n")
				b.WriteString(m.runLine(run))
				if child := m.childLine(run); child != "" {
					fmt.Fprintf(&b, "%s%s\n", childIndent, child)
				}
			}
		}
	}

	return pinFooter(b.String(), gutter+m.styles.muted.Render(m.footer), m.height)
}

func (m watchModel) header() string {
	left := gutter + m.styles.title.Render("patchdock")
	tally := m.styles.muted.Render(m.tally())

	gap := m.width - ansi.StringWidth(left) - ansi.StringWidth(tally) - len(gutter)
	if gap < 2 {
		return left + "  " + tally
	}
	return left + strings.Repeat(" ", gap) + tally
}

func (m watchModel) runLine(run api.Run) string {
	left := fmt.Sprintf("%s%s %s  ",
		subIndent,
		m.mark(run.Status),
		m.statusWord(run.Status))

	attempt := ""
	if run.Attempt > 1 {
		attempt = m.styles.muted.Render(fmt.Sprintf("  attempt %d", run.Attempt))
	}

	right := m.styles.muted.Render(short(elapsed(run, time.Now())))

	avail := m.width - ansi.StringWidth(left) - ansi.StringWidth(attempt) -
		ansi.StringWidth(right) - 2 - len(gutter)
	title := ansi.Truncate(oneLine(run.Title), max(avail, minDetail), "…")

	line := left + title + attempt
	gap := max(m.width-ansi.StringWidth(line)-ansi.StringWidth(right)-len(gutter), 2)
	return line + strings.Repeat(" ", gap) + right + "\n"
}

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

const statusWidth = len(api.StatusPublishing)

func (m watchModel) statusWord(status api.Status) string {
	word := pad(string(status), statusWidth)

	switch status {
	case api.StatusQueued, api.StatusSucceeded:
		return m.styles.muted.Render(word)
	case api.StatusFailed, api.StatusRejected:
		return m.styles.red.Render(word)
	case api.StatusCancelled:
		return m.styles.amber.Render(word)
	default:
		return m.styles.accent.Render(word)
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
