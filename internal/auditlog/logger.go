package auditlog

import (
	"bytes"
	"encoding/json"
	"errors"
	"fmt"
	"os"
	"path/filepath"

	"github.com/HJyup/patchdock/internal/stage"
)

// stagesDir groups the per-stage contract folders under the run dir
const stagesDir = "stages"

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

func (l *Logger) WriteDiffs(diffs []byte) error {
	if l.logDir == "" {
		return fmt.Errorf("cannot write patch: log directory is not initialized")
	}

	diffsPath := filepath.Join(l.logDir, "workspace.patch")
	if err := os.WriteFile(diffsPath, diffs, 0o644); err != nil {
		return fmt.Errorf("failed to write workspace.patch: %w", err)
	}

	return nil
}

func (l *Logger) ArchiveStage(srcDir string) error {
	if l.logDir == "" {
		return fmt.Errorf("cannot archive stage: log directory is not initialized")
	}

	label := filepath.Base(srcDir)
	dstDir := filepath.Join(l.logDir, stagesDir, label)
	if err := os.MkdirAll(dstDir, 0o755); err != nil {
		return fmt.Errorf("create stage log dir %s: %w", label, err)
	}

	for _, name := range []string{stage.Input, stage.Output} {
		data, err := os.ReadFile(filepath.Join(srcDir, name))
		if errors.Is(err, os.ErrNotExist) {
			continue
		}
		if err != nil {
			return fmt.Errorf("read %s/%s: %w", label, name, err)
		}

		out := data
		var pretty bytes.Buffer
		if json.Indent(&pretty, data, "", "  ") == nil {
			out = pretty.Bytes()
		}

		if err := os.WriteFile(filepath.Join(dstDir, name), out, 0o644); err != nil {
			return fmt.Errorf("archive %s/%s: %w", label, name, err)
		}
	}

	return nil
}

func (l *Logger) Close() error {
	if l.logFile != nil {
		return l.logFile.Close()
	}
	return nil
}
