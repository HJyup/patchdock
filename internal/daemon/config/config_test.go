package config

import (
	"os"
	"path/filepath"
	"strings"
	"testing"
	"time"
)

func configPath(t *testing.T) string {
	t.Helper()
	return filepath.Join(t.TempDir(), "config.json")
}

func writeConfig(t *testing.T, contents string) string {
	t.Helper()

	path := configPath(t)
	if err := os.WriteFile(path, []byte(contents), 0o600); err != nil {
		t.Fatalf("write config: %v", err)
	}
	return path
}

func mustLoad(t *testing.T, path string) *Config {
	t.Helper()

	cfg, err := Load(path)
	if err != nil {
		t.Fatalf("load config: %v", err)
	}
	return cfg
}

func TestDefaultsAreValid(t *testing.T) {
	cfg := Defaults()

	if err := cfg.Validate(); err != nil {
		t.Fatalf("Defaults() does not satisfy Validate: %v", err)
	}
}

func TestLoadCreatesTheFileWhenMissing(t *testing.T) {
	path := configPath(t)

	cfg := mustLoad(t, path)

	if *cfg != Defaults() {
		t.Errorf("config = %+v, want %+v", *cfg, Defaults())
	}

	info, err := os.Stat(path)
	if err != nil {
		t.Fatalf("config file was not created: %v", err)
	}

	// The runtime directory is per-user and this file configures a daemon, so
	// it must not be world- or group-readable.
	if perm := info.Mode().Perm(); perm != 0o600 {
		t.Errorf("permissions = %04o, want 0600", perm)
	}
}

func TestCreatedFileWritesDurationsAsStrings(t *testing.T) {
	path := configPath(t)
	mustLoad(t, path)

	raw, err := os.ReadFile(path)
	if err != nil {
		t.Fatalf("read config: %v", err)
	}
	contents := string(raw)

	for _, want := range []string{`"retention": "15m0s"`, `"snapshot_tick": "200ms"`} {
		if !strings.Contains(contents, want) {
			t.Errorf("config file does not contain %s\ngot:\n%s", want, contents)
		}
	}
}

func TestCreatedFileReloadsUnchanged(t *testing.T) {
	path := configPath(t)

	created := mustLoad(t, path)
	reloaded := mustLoad(t, path)

	if *created != *reloaded {
		t.Errorf("reloaded = %+v, want %+v", *reloaded, *created)
	}
}

func TestLoadParsesDurationStrings(t *testing.T) {
	path := writeConfig(t, `{"retention": "90s", "snapshot_tick": "1s"}`)

	cfg := mustLoad(t, path)

	if got := cfg.Retention.Duration(); got != 90*time.Second {
		t.Errorf("retention = %v, want 90s", got)
	}
	if got := cfg.SnapshotTick.Duration(); got != time.Second {
		t.Errorf("snapshot_tick = %v, want 1s", got)
	}
}

func TestLoadRejectsUnknownFields(t *testing.T) {
	path := writeConfig(t, `{"max_container": 7}`)

	_, err := Load(path)
	if err == nil {
		t.Fatal("a misspelled field was accepted")
	}
	if !strings.Contains(err.Error(), "max_container") {
		t.Errorf("error = %v, want it to name the unknown field", err)
	}
}

func TestLoadRejectsBadInput(t *testing.T) {
	tests := []struct {
		name     string
		contents string
		wantErr  string
	}{
		{
			name:     "empty file",
			contents: "",
			wantErr:  "delete it to recreate it with defaults",
		},
		{
			name:     "truncated json",
			contents: `{"max_containers":`,
			wantErr:  "decode config",
		},
		{
			name:     "wrong type for a duration",
			contents: `{"retention": true}`,
			wantErr:  "decode config",
		},
		{
			name:     "unparseable duration",
			contents: `{"retention": "soon"}`,
			wantErr:  "decode config",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			path := writeConfig(t, tt.contents)

			_, err := Load(path)
			if err == nil {
				t.Fatalf("Load(%q) succeeded, want an error", tt.contents)
			}
			if !strings.Contains(err.Error(), tt.wantErr) {
				t.Errorf("error = %v, want it to contain %q", err, tt.wantErr)
			}
		})
	}
}

func TestValidateReportsEveryViolationAtOnce(t *testing.T) {
	cfg := Config{}

	err := cfg.Validate()
	if err == nil {
		t.Fatal("the zero config was accepted")
	}

	for _, want := range []string{
		"config.max_containers",
		"config.retention",
		"config.inbox_size",
		"config.snapshot_tick",
	} {
		if !strings.Contains(err.Error(), want) {
			t.Errorf("error is missing %s\ngot: %v", want, err)
		}
	}
}

func TestLoadFailsWhenTheDirectoryDoesNotExist(t *testing.T) {
	path := filepath.Join(t.TempDir(), "missing-dir", "config.json")

	if _, err := Load(path); err == nil {
		t.Fatal("Load succeeded with no directory to write into")
	}
}
