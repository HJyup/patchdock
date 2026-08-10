// Package tui renders patchdock's terminal surfaces: the task prompt, the
// post-submit choice, and the watch dashboard that follows every run the
// daemon reports. On anything that is not a terminal the interactive surfaces
// refuse to open rather than degrade into escape sequences
package tui

import (
	"fmt"
	"io"
	"os"
	"strings"
	"time"
	"unicode"

	"github.com/charmbracelet/x/ansi"
)

const (
	// gutter is the left margin every line is printed against
	gutter = "  "
	// subIndent nests the lines belonging to a block's own title
	subIndent   = gutter + "  "
	childIndent = gutter + "    "

	fallbackCols = 80
	// minDetail keeps truncated lines legible on an implausibly narrow terminal
	minDetail = 20
	timeWidth = 6
)

const (
	sucessSign = "✔"
	rejectSign = "✖"
	arrowSign  = "→"
	queuedSign = "•"
)

func Interactive(in io.Reader, out io.Writer) bool {
	return usable(in) && usable(out)
}

func usable(stream any) bool {
	return isTerminal(stream) && os.Getenv("TERM") != "dumb"
}

func isTerminal(stream any) bool {
	file, ok := stream.(*os.File)
	if !ok {
		return false
	}

	info, err := file.Stat()
	if err != nil {
		return false
	}

	return info.Mode()&os.ModeCharDevice != 0
}

func short(d time.Duration) string {
	d = d.Round(time.Second)
	if d < time.Minute {
		return fmt.Sprintf("%ds", int(d.Seconds()))
	}
	return fmt.Sprintf("%dm%02ds", int(d.Minutes()), int(d.Seconds())%60)
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
