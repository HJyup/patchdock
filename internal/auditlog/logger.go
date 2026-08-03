package auditlog

import (
	"encoding/json"
	"errors"
	"fmt"
	"os"
	"path/filepath"
)

const (
	summaryFile      = "run.md"
	recordFile       = "run.json"
	streamFile       = "stdout.log"
	failedOutputFile = "failed-output.json"
)

type Logger struct {
	LogID         string // still don't know whether it's the best way to indicate, but since we have logger per pipeline, it can work
	LogDir        string
	logStreamFile *os.File
}

func New(id string, dir string) (*Logger, error) {
	logDir := filepath.Join(dir, "logs", id)

	if err := os.MkdirAll(logDir, 0o755); err != nil {
		return nil, fmt.Errorf("failed creating log directory: %w", err)
	}

	logPath := filepath.Join(logDir, streamFile)
	file, err := os.OpenFile(logPath, os.O_CREATE|os.O_WRONLY|os.O_APPEND, 0o644)
	if err != nil {
		return nil, fmt.Errorf("failed creating log file: %w", err)
	}

	return &Logger{
		LogDir:        filepath.Join(dir, "logs", id),
		LogID:         id,
		logStreamFile: file,
	}, nil
}

func (l *Logger) Write(p []byte) (n int, err error) {
	n, err = l.logStreamFile.Write(p)
	if err != nil {
		return n, fmt.Errorf("log write error: %w", err)
	}

	return n, nil
}

func (l *Logger) WriteRun(rec *Record) error {
	encoded, err := json.MarshalIndent(rec, "", "  ")
	if err != nil {
		return fmt.Errorf("encode %s: %w", recordFile, err)
	}
	encoded = append(encoded, '\n')

	return errors.Join(
		l.writeFile(summaryFile, renderRun(rec)),
		l.writeFile(recordFile, encoded),
	)
}

func (l *Logger) WriteFailedOutput(raw []byte) error {
	return l.writeFile(failedOutputFile, raw)
}

func (l *Logger) writeFile(name string, content []byte) error {
	if err := os.WriteFile(filepath.Join(l.LogDir, name), content, 0o644); err != nil {
		return fmt.Errorf("failed to write %s: %w", name, err)
	}
	return nil
}

func (l *Logger) Close() error {
	if l.logStreamFile != nil {
		return l.logStreamFile.Close()
	}
	return nil
}
