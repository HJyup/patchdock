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
	patchFile        = "workspace.patch"
	streamFile       = "stdout.log"
	failedOutputFile = "failed-output.json"
)

type Logger struct {
	logDir  string
	logFile *os.File
}

func New(logDir string) (*Logger, error) {
	if err := os.MkdirAll(logDir, 0o755); err != nil {
		return nil, fmt.Errorf("failed creating log directory: %w", err)
	}

	logPath := filepath.Join(logDir, streamFile)
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

func (l *Logger) WriteRun(rec *Record) error {
	if l.logDir == "" {
		return fmt.Errorf("cannot write run record: log directory is not initialized")
	}

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

func (l *Logger) WritePatch(diff string) error {
	return l.writeFile(patchFile, []byte(diff))
}

func (l *Logger) WriteFailedOutput(raw []byte) error {
	return l.writeFile(failedOutputFile, raw)
}

func (l *Logger) writeFile(name string, content []byte) error {
	if l.logDir == "" {
		return fmt.Errorf("cannot write %s: log directory is not initialized", name)
	}
	if err := os.WriteFile(filepath.Join(l.logDir, name), content, 0o644); err != nil {
		return fmt.Errorf("failed to write %s: %w", name, err)
	}
	return nil
}

func (l *Logger) Close() error {
	if l.logFile != nil {
		return l.logFile.Close()
	}
	return nil
}
