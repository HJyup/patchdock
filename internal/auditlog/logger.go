package auditlog

import (
	"fmt"
	"os"
	"path/filepath"
)

type Logger struct {
	logDir  string
	logFile *os.File
}

func NewLogger(logDir string) (*Logger, error) {
	if err := os.MkdirAll(logDir, 0o755); err != nil {
		return nil, fmt.Errorf("failed creating log directory: %w", err)
	}

	logPath := filepath.Join(logDir, "stdout.log")
	file, err := os.OpenFile(logPath, os.O_CREATE|os.O_WRONLY|os.O_APPEND, 0o644)
	if err != nil {
		return nil, fmt.Errorf("failed creating log file: %w", err)
	}

	return &Logger{
		logDir:  logDir,
		logFile: file,
	}, nil
}

func (l *Logger) Write(p []byte) (n int, err error) {
	if l.logFile == nil {
		return 0, fmt.Errorf("cannot write: log file descriptor is not open")
	}

	n, err = l.logFile.Write(p)
	if err != nil {
		return n, fmt.Errorf("log write error: %w", err)
	}

	return n, nil
}

func (l *Logger) WriteOutcome(outcome []byte) error {
	if l.logDir == "" {
		return fmt.Errorf("cannot write outcome: log directory is not initialized")
	}

	outputPath := filepath.Join(l.logDir, "outcome.json")
	if err := os.WriteFile(outputPath, outcome, 0644); err != nil {
		return fmt.Errorf("failed to write outcome.json: %w", err)
	}

	return nil
}

func (l *Logger) WriteDiffs(diffs []byte) error {
	if l.logDir == "" {
		return fmt.Errorf("cannot write outcome: log directory is not initialized")
	}

	diffsPath := filepath.Join(l.logDir, "workspace.patch")
	if err := os.WriteFile(diffsPath, diffs, 0644); err != nil {
		return fmt.Errorf("failed to write outcome.json: %w", err)
	}

	return nil
}

func (l *Logger) Close() error {
	if l.logFile != nil {
		return l.logFile.Close()
	}
	return nil
}
