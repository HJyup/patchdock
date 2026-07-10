package config

import (
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

func TestValidateTranslatesFieldErrors(t *testing.T) {
	tests := []struct {
		name   string
		mutate func(*Config)
		want   string
	}{
		{
			name:   "missing stages reports each stage sorted",
			mutate: func(c *Config) { c.Stages = nil },
			want: "config.stages.executor: missing\n" +
				"config.stages.planner: missing\n" +
				"config.stages.reviewer: missing",
		},
		{
			name:   "stage file must be typescript",
			mutate: func(c *Config) { c.Stages[types.StagePlanner] = "planner.js" },
			want:   "config.stages[planner]: must be a .ts file",
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
