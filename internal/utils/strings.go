package utils

import (
	"fmt"
	"strings"
)

const maxSubjectLine = 72

func FirstLine(s string) string {
	line, _, _ := strings.Cut(strings.TrimSpace(s), "\n")
	line = strings.TrimSpace(line)

	runes := []rune(line)
	if len(runes) <= maxSubjectLine {
		return line
	}

	return string(runes[:maxSubjectLine-1]) + "…"
}

func Plural(n int, noun string) string {
	if n == 1 {
		return fmt.Sprintf("%d %s", n, noun)
	}
	return fmt.Sprintf("%d %ss", n, noun)
}
