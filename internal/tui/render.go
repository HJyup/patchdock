package tui

import (
	"fmt"
	"strings"
	"time"

	"github.com/charmbracelet/x/ansi"
)

const (
	// timeWidth holds the longest elapsed time short() produces before a run
	// would have hit any sane container timeout
	timeWidth = 6
	// limitPressure is the fraction of a limit past which the figure stops being
	// background information and starts being a warning
	limitPressure = 0.8
)

// headerLines renders the run's identity block. Only the task line is
// truncated: the repo and the log path are the two things a reader may need to
// copy, so they are never clipped
func headerLines(info RunInfo, s styles, width int) []string {
	title := gutter + s.title.Render(info.Repo)
	if info.RunID != "" {
		title += "  " + s.muted.Render(info.RunID)
	}

	lines := []string{title}

	if info.Task != "" {
		task := ansi.Truncate(info.Task, max(width-len(subIndent), minDetail), "…")
		lines = append(lines, subIndent+task)
	}
	if info.LogDir != "" {
		lines = append(lines, subIndent+s.muted.Render("logs "+arrowSign+" "+info.LogDir))
	}

	return lines
}

// summaryLines renders the closing account of a run: the verdict, what it cost,
// what it produced, and the one command that picks the work up
func summaryLines(res Result, s styles) []string {
	headline, tone := "Accepted", s.green
	if !res.Accepted {
		headline, tone = "Rejected", s.red
	}

	lines := []string{fmt.Sprintf("%s%s  %s",
		gutter, tone.Bold(true).Render(headline), s.muted.Render(short(res.Duration)))}

	if stat := diffStat(res, s); stat != "" {
		if res.Branch != "" {
			stat += s.muted.Render("  "+arrowSign+"  ") + res.Branch
		} else {
			stat += s.muted.Render("  (nothing published)")
		}
		lines = append(lines, subIndent+stat)
	}

	if res.Branch != "" {
		lines = append(lines, subIndent+s.accent.Render("git fetch && git switch "+res.Branch))
	}
	if res.LogDir != "" {
		lines = append(lines, subIndent+s.muted.Render("logs "+arrowSign+" "+res.LogDir))
	}

	return lines
}

// diffStat drops a side that is zero: "+51" says everything "+51 -0" does, and
// the absent half is one less figure to read past
func diffStat(res Result, s styles) string {
	if res.Files == 0 {
		return ""
	}

	churn := make([]string, 0, 2)
	if res.Additions > 0 {
		churn = append(churn, s.green.Render(fmt.Sprintf("+%d", res.Additions)))
	}
	if res.Deletions > 0 {
		churn = append(churn, s.red.Render(fmt.Sprintf("-%d", res.Deletions)))
	}

	stat := plural(res.Files, "file")
	if len(churn) > 0 {
		stat += "  " + strings.Join(churn, " ")
	}

	return stat
}

func remaining(s styles, timeout time.Duration, elapsed time.Duration) string {
	if timeout <= 0 {
		return ""
	}

	return s.pressure(float64(elapsed), float64(timeout)).Render(coarse(timeout-elapsed) + " left")
}

// coarse rounds a remaining duration to the unit a reader would act on. A
// countdown to the second invites watching it rather than the run
func coarse(d time.Duration) string {
	switch {
	case d <= 0:
		return "0s"
	case d < time.Minute:
		return fmt.Sprintf("%ds", int(d.Seconds()))
	default:
		return fmt.Sprintf("%dm", int(d.Minutes()))
	}
}

// oneLine flattens prose onto a single row. A plan or a review verdict is
// written as sentences and may wrap, and a row that silently becomes three
// pushes everything below it around as the run progresses
func oneLine(text string) string {
	return strings.TrimSpace(strings.ReplaceAll(text, "\n", " "))
}

func pad(text string, width int) string {
	if gap := width - ansi.StringWidth(text); gap > 0 {
		return text + strings.Repeat(" ", gap)
	}
	return text
}

func plural(n int, noun string) string {
	if n == 1 {
		return fmt.Sprintf("%d %s", n, noun)
	}
	return fmt.Sprintf("%d %ss", n, noun)
}
