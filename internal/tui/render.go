package tui

import (
	"fmt"
	"strings"
	"time"
	"unicode"

	"github.com/charmbracelet/x/ansi"
)

const (
	timeWidth     = 6
	limitPressure = 0.8
)

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

func oneLine(text string) string {
	var b strings.Builder
	b.Grow(len(text))

	space := true // leading whitespace is skipped by starting "inside" a run
	for _, r := range text {
		if r == '\t' || r == '\n' || r == '\r' || unicode.IsSpace(r) || unicode.IsControl(r) {
			if !space {
				b.WriteByte(' ')
				space = true
			}
			continue
		}
		b.WriteRune(r)
		space = false
	}

	return strings.TrimRight(b.String(), " ")
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
