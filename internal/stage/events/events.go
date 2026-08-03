// Package events turns one container run into the structured log stream that
// Patchdock records and displays. It owns the event vocabulary shared with the
// SDK - "stage_started", "command_completed" and the rest - so the stage runner
// only has to hand it container output
package events

import (
	"encoding/json"
	"fmt"
	"io"
	"time"

	"github.com/HJyup/patchdock/internal/docker"
	"github.com/HJyup/patchdock/internal/types"
)

const sourcePatchdock = "patchdock"
const sourceContainer = "container"

type Stream struct {
	out      io.Writer
	stage    types.StageName
	activity func(string)
}

func New(out io.Writer, stage types.StageName, activity func(string)) *Stream {
	if out == nil {
		out = io.Discard
	}
	if activity == nil {
		activity = func(string) {}
	}

	return &Stream{out: out, stage: stage, activity: activity}
}

func (s *Stream) Started() error {
	return s.write(map[string]any{
		"source": sourcePatchdock,
		"event":  "stage_started",
	})
}

func (s *Stream) Line(msg docker.LogLine) error {
	event := make(map[string]any)
	if err := json.Unmarshal([]byte(msg.Text), &event); err != nil {
		event = map[string]any{
			"source":  sourceContainer,
			"event":   "message",
			"message": msg.Text,
		}
	}

	if activity := activityOf(event); activity != "" {
		s.activity(activity)
	}

	event["stream"] = msg.Stream
	return s.write(event)
}

func (s *Stream) Finished(res docker.RunResult) error {
	event := map[string]any{
		"source":    sourcePatchdock,
		"event":     "stage_finished",
		"exit_code": res.ExitCode,
		"level":     "info",
	}
	if res.Err != nil {
		event["level"] = "error"
		event["error"] = res.Err.Error()
	} else if res.ExitCode != 0 {
		event["level"] = "error"
	}

	return s.write(event)
}

func (s *Stream) write(event map[string]any) error {
	event["stage"] = s.stage
	if _, exists := event["timestamp"]; !exists {
		event["timestamp"] = time.Now().UTC().Format(time.RFC3339Nano)
	}

	line, err := json.Marshal(event)
	if err != nil {
		return fmt.Errorf("events: encode log event: %w", err)
	}
	if _, err := fmt.Fprintln(s.out, string(line)); err != nil {
		return fmt.Errorf("events: failed writing to log stream: %w", err)
	}

	return nil
}
