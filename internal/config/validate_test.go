package config

import (
	"os"
	"path/filepath"
	"testing"

	"github.com/HJyup/patchdock/internal/types"
)

func validStages() map[types.StageName]string {
	return map[types.StageName]string{
		types.StagePlanner:  "planner.ts",
		types.StageExecutor: "executor.ts",
		types.StageReviewer: "reviewer.ts",
	}
}

func TestValidateAcceptsDefaultsWithStages(t *testing.T) {
	cfg := Defaults()
	cfg.Stages = validStages()
	if err := cfg.Validate(); err != nil {
		t.Fatalf("valid config rejected: %v", err)
	}
}

func TestValidateAcceptsOptionalCodexConfig(t *testing.T) {
	cfg := Defaults()
	cfg.Stages = validStages()
	cfg.Codex = &CodexConfig{Auth: CodexHostLogin}

	if err := cfg.Validate(); err != nil {
		t.Fatalf("valid config rejected: %v", err)
	}
}

func TestLoadCodexConfigIsOptional(t *testing.T) {
	tests := []struct {
		name      string
		codexYAML string
		wantCodex *CodexConfig
	}{
		{name: "omitted"},
		{
			name:      "host login",
			codexYAML: "codex:\n  auth: host-login\n",
			wantCodex: &CodexConfig{Auth: CodexHostLogin},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			path := filepath.Join(t.TempDir(), "config.yml")
			content := tt.codexYAML + "stages:\n" +
				"  planner: planner.ts\n" +
				"  executor: executor.ts\n" +
				"  reviewer: reviewer.ts\n"
			if err := os.WriteFile(path, []byte(content), 0o600); err != nil {
				t.Fatalf("write config: %v", err)
			}

			cfg, err := Load(path)
			if err != nil {
				t.Fatalf("load config: %v", err)
			}
			if tt.wantCodex == nil {
				if cfg.Codex != nil {
					t.Fatalf("unexpected Codex config: %#v", cfg.Codex)
				}
				return
			}
			if cfg.Codex == nil || cfg.Codex.Auth != tt.wantCodex.Auth {
				t.Fatalf("Codex config mismatch: got %#v, want %#v", cfg.Codex, tt.wantCodex)
			}
		})
	}
}

func TestValidateFieldErrors(t *testing.T) {
	tests := []struct {
		name   string
		mutate func(*Config)
		want   string
	}{
		{
			name:   "missing stages reports each stage in pipeline order",
			mutate: func(c *Config) { c.Stages = nil },
			want: "config.stages[planner]: missing\n" +
				"config.stages[executor]: missing\n" +
				"config.stages[reviewer]: missing",
		},
		{
			name:   "stage file must be typescript",
			mutate: func(c *Config) { c.Stages[types.StagePlanner] = "planner.js" },
			want:   "config.stages[planner]: must be a .ts file",
		},
		{
			name:   "stage file must not be empty",
			mutate: func(c *Config) { c.Stages[types.StageReviewer] = "" },
			want:   "config.stages[reviewer]: empty",
		},
		{
			name:   "unknown stage keys are rejected",
			mutate: func(c *Config) { c.Stages["deployer"] = "deployer.ts" },
			want:   "config.stages[deployer]: unknown stage",
		},
		{
			name:   "missing Codex auth",
			mutate: func(c *Config) { c.Codex = &CodexConfig{} },
			want:   "config.codex.auth: missing",
		},
		{
			name:   "unsupported Codex auth",
			mutate: func(c *Config) { c.Codex = &CodexConfig{Auth: "unknown"} },
			want:   "config.codex.auth: unsupported value \"unknown\"",
		},
		{
			name:   "negative timeout",
			mutate: func(c *Config) { c.Container.Timeout = Duration(-1) },
			want:   "config.container.timeout: must be >= 0",
		},
		{
			name:   "negative retries",
			mutate: func(c *Config) { c.Retries.Max = -1 },
			want:   "config.retries.max: must be >= 0",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			cfg := Defaults()
			cfg.Stages = validStages()
			tt.mutate(&cfg)

			err := cfg.Validate()
			if err == nil {
				t.Fatal("expected a validation error, got nil")
			}
			if err.Error() != tt.want {
				t.Fatalf("error mismatch\n got: %q\nwant: %q", err.Error(), tt.want)
			}
		})
	}
}
