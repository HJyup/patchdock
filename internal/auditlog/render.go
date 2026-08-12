package auditlog

import (
	"fmt"
	"strings"

	"github.com/HJyup/patchdock/internal/utils"
)

func renderRun(rec *Record) []byte {
	var b strings.Builder

	fmt.Fprintf(&b, "# %s · %s · %s · %s\n\n",
		rec.RunID, outcomeWord(rec), utils.Plural(len(rec.Attempts), "attempt"), rec.Duration)

	if title := strings.TrimSpace(rec.Task.Title); title != "" {
		fmt.Fprintf(&b, "**%s**\n\n", title)
	}
	fmt.Fprintf(&b, "**Task:** %s\n\n", strings.TrimSpace(rec.Task.Description))

	if rec.Plan.Summary != "" {
		b.WriteString("## Plan\n\n")
		fmt.Fprintf(&b, "%s\n\n", strings.TrimSpace(rec.Plan.Summary))
		if body := strings.TrimSpace(rec.Plan.Body); body != "" {
			fmt.Fprintf(&b, "%s\n\n", nest(body))
		}
	}

	for _, attempt := range rec.Attempts {
		renderAttempt(&b, attempt)
	}

	if rec.Failure != nil {
		b.WriteString("## Failure\n\n")
		fmt.Fprintf(&b, "**%s** — %s\n\n", rec.Failure.Stage, strings.TrimSpace(rec.Failure.Message))
		if rec.Failure.RawOutput {
			fmt.Fprintf(&b, "The output the stage actually wrote is in `%s`.\n\n", failedOutputFile)
		}
	}

	b.WriteString("---\n")
	b.WriteString(renderFooter(rec))

	return []byte(b.String())
}

func renderAttempt(b *strings.Builder, attempt Attempt) {
	decision := string(attempt.Review.Decision)
	if decision == "" {
		decision = "no review"
	}
	fmt.Fprintf(b, "## Attempt %d — %s → %s\n\n", attempt.Number, attempt.Execution.Status, decision)

	if notes := strings.TrimSpace(attempt.Execution.Notes); notes != "" {
		fmt.Fprintf(b, "%s\n\n", nest(notes))
	} else {
		b.WriteString("_The executor recorded no notes._\n\n")
	}

	if attempt.Review.Decision == "" {
		return
	}

	fmt.Fprintf(b, "> **Reviewer:** %s\n\n", strings.Join(strings.Fields(attempt.Review.Summary), " "))
	if feedback := strings.TrimSpace(attempt.Review.Feedback); feedback != "" {
		fmt.Fprintf(b, "%s\n\n", nest(feedback))
	}
}

func renderFooter(rec *Record) string {
	if rec.Patch.Files == 0 {
		return "No file changes · raw container stream in `stdout.log`\n"
	}
	return fmt.Sprintf(" %s, +%d -%d · raw container stream in `stdout.log`\n",
		utils.Plural(rec.Patch.Files, "file"), rec.Patch.Additions, rec.Patch.Deletions)
}

func outcomeWord(rec *Record) string {
	switch {
	case rec.Failure != nil:
		return "failed"
	case rec.Accepted:
		return "accepted"
	default:
		return "rejected"
	}
}

const sectionDepth = 2

func nest(md string) string {
	shallowest := 7
	eachLine(md, func(line string, level int) {
		if level > 0 && level < shallowest {
			shallowest = level
		}
	})

	shift := sectionDepth + 1 - shallowest
	if shift <= 0 {
		return md
	}

	var out []string
	eachLine(md, func(line string, level int) {
		if level > 0 && level+shift <= 6 {
			line = strings.Repeat("#", shift) + line
		}
		out = append(out, line)
	})

	return strings.Join(out, "\n")
}

func eachLine(md string, visit func(line string, level int)) {
	fence := ""

	for line := range strings.SplitSeq(md, "\n") {
		trimmed := strings.TrimSpace(line)

		switch {
		case fence != "":
			if strings.HasPrefix(trimmed, fence) {
				fence = ""
			}
		case strings.HasPrefix(trimmed, "```"):
			fence = "```"
		case strings.HasPrefix(trimmed, "~~~"):
			fence = "~~~"
		default:
			level := len(trimmed) - len(strings.TrimLeft(trimmed, "#"))
			if level > 0 && level <= 6 && strings.HasPrefix(trimmed[level:], " ") {
				visit(line, level)
				continue
			}
		}

		visit(line, 0)
	}
}
