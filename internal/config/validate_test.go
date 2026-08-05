package config

import (
	"os"
	"path/filepath"
	"reflect"
	"testing"

	"github.com/HJyup/patchdock/internal/types"
)

const namespaceError = "config.name_space: must be lowercase letters or digits, separated by '.', '_' or '-'"

func validStages() map[types.StageName]string {
	return map[types.StageName]string{
		types.StagePlanner:  "planner.ts",
		types.StageExecutor: "executor.ts",
		types.StageReviewer: "reviewer.ts",
	}
}

func TestValidateAcceptsDefaultsWithStages(t *testing.T) {
	cfg := Defaults()
	cfg.Namespace = "test-ns"
	cfg.Stages = validStages()
	if err := cfg.Validate(); err != nil {
		t.Fatalf("valid config rejected: %v", err)
	}
}

func TestValidateAcceptsCredentials(t *testing.T) {
	cfg := Defaults()
	cfg.Namespace = "test-ns"
	cfg.Stages = validStages()
	cfg.Credentials = []Credential{{
		Host:   "~/.codex/auth.json",
		Target: "/codex-auth/auth.json",
		Env:    map[string]string{"CODEX_HOME": "/codex-auth"},
	}}

	if err := cfg.Validate(); err != nil {
		t.Fatalf("valid config rejected: %v", err)
	}
}

func TestLoadCredentialsAreOptional(t *testing.T) {
	tests := []struct {
		name  string
		yaml  string
		want  []Credential
		count int
	}{
		{name: "omitted"},
		{
			name: "host file with env",
			yaml: "credentials:\n" +
				"  - host: ~/.codex/auth.json\n" +
				"    target: /codex-auth/auth.json\n" +
				"    env:\n" +
				"      CODEX_HOME: /codex-auth\n",
			want: []Credential{{
				Host:   "~/.codex/auth.json",
				Target: "/codex-auth/auth.json",
				Env:    map[string]string{"CODEX_HOME": "/codex-auth"},
			}},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			path := filepath.Join(t.TempDir(), "config.yml")
			content := "name_space: test-ns\n" + tt.yaml + "stages:\n" +
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
			if !reflect.DeepEqual(cfg.Credentials, tt.want) {
				t.Fatalf("credentials mismatch\n got: %#v\nwant: %#v", cfg.Credentials, tt.want)
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
			name:   "credential without a host",
			mutate: func(c *Config) { c.Credentials = []Credential{{Target: "/codex-auth/auth.json"}} },
			want:   "config.credentials[0].host: missing",
		},
		{
			name:   "credential without a target",
			mutate: func(c *Config) { c.Credentials = []Credential{{Host: "~/.codex/auth.json"}} },
			want:   "config.credentials[0].target: missing",
		},
		{
			name: "credential target must be absolute",
			mutate: func(c *Config) {
				c.Credentials = []Credential{{Host: "~/.codex/auth.json", Target: "codex-auth/auth.json"}}
			},
			want: "config.credentials[0].target: must be an absolute container path",
		},
		{
			name: "credential targets must be unique",
			mutate: func(c *Config) {
				c.Credentials = []Credential{
					{Host: "~/.codex/auth.json", Target: "/codex-auth/auth.json"},
					{Host: "~/.other/auth.json", Target: "/codex-auth/auth.json"},
				}
			},
			want: "config.credentials[1].target: \"/codex-auth/auth.json\" is already mounted",
		},
		{
			name: "credential env names must not be empty",
			mutate: func(c *Config) {
				c.Credentials = []Credential{{
					Host:   "~/.codex/auth.json",
					Target: "/codex-auth/auth.json",
					Env:    map[string]string{"": "/codex-auth"},
				}}
			},
			want: "config.credentials[0].env: empty variable name",
		},
		{
			name:   "negative timeout",
			mutate: func(c *Config) { c.Container.Timeout = Duration(-1) },
			want:   "config.container.timeout: must be >= 0",
		},
		{
			name:   "missing namespace",
			mutate: func(c *Config) { c.Namespace = "" },
			want:   namespaceError,
		},
		{
			name:   "namespace with uppercase or spaces",
			mutate: func(c *Config) { c.Namespace = "My Name" },
			want:   namespaceError,
		},
		{
			name:   "negative retries",
			mutate: func(c *Config) { c.Retries.Max = -1 },
			want:   "config.retries.max: must be >= 1",
		},
		{
			name:   "zero retries runs no attempts",
			mutate: func(c *Config) { c.Retries.Max = 0 },
			want:   "config.retries.max: must be >= 1",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			cfg := Defaults()
			cfg.Namespace = "test-ns"
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
