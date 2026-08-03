package events

import (
	"fmt"
	"strings"
)

// Should be moved somewhere, since we define it here but it's mainly the problem of TUI
const maxActivity = 140

func activityOf(event map[string]any) string {
	switch text(event["event"]) {
	case "process_started":
		return "starting agent"

	case "session_started":
		return "thinking"

	case "command_completed":
		if command := text(event["command"]); command != "" {
			return clip("$ " + command)
		}
		return "ran a command"

	case "file_change_completed":
		return changeSummary(event["changes"])

	case "tool_call_completed":
		server, tool := text(event["server"]), text(event["tool"])
		switch {
		case server != "" && tool != "":
			return clip(server + "." + tool)
		case tool != "":
			return clip(tool)
		}
		return "called a tool"

	case "turn_completed":
		return "thinking"

	case "message":
		return clip(text(event["message"]))
	}

	return ""
}

func changeSummary(raw any) string {
	changes, ok := raw.([]any)
	if !ok || len(changes) == 0 {
		return "edited files"
	}

	first := ""
	if entry, ok := changes[0].(map[string]any); ok {
		first = text(entry["path"])
	}
	if first == "" {
		return fmt.Sprintf("edited %d files", len(changes))
	}

	if len(changes) == 1 {
		return clip("edited " + first)
	}

	return clip(fmt.Sprintf("edited %s (+%d more)", first, len(changes)-1))
}

func text(value any) string {
	str, ok := value.(string)
	if !ok {
		return ""
	}

	return strings.TrimSpace(strings.ReplaceAll(str, "\n", " "))
}

func clip(s string) string {
	runes := []rune(s)
	if len(runes) <= maxActivity {
		return s
	}

	return string(runes[:maxActivity-1]) + "…"
}
