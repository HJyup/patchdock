package config

import (
	"errors"
	"fmt"
	"path/filepath"
	"slices"
	"strings"

	"github.com/HJyup/patchdock/internal/types"
)

var requiredStages = []types.StageName{types.StagePlanner, types.StageExecutor, types.StageReviewer}

func (c *Config) Validate() error {
	var errs []error
	addf := func(format string, args ...any) {
		errs = append(errs, fmt.Errorf(format, args...))
	}

	if c.Container.Timeout < 0 {
		addf("config.container.timeout: must be >= 0")
	}
	if c.Container.TokenBudget < 0 {
		addf("config.container.token_budget: must be >= 0")
	}
	if c.Container.MaxContainers < 0 {
		addf("config.container.max_containers: must be >= 0")
	}
	if c.Retries.Max < 0 {
		addf("config.retries.max: must be >= 0")
	}
	if c.Codex != nil {
		switch c.Codex.Auth {
		case "":
			addf("config.codex.auth: missing")
		case CodexHostLogin:
		default:
			addf("config.codex.auth: unsupported value %q", c.Codex.Auth)
		}
	}

	for _, stage := range requiredStages {
		file, ok := c.Stages[stage]
		if !ok {
			addf("config.stages[%s]: missing", stage)
			continue
		}
		if file == "" {
			addf("config.stages[%s]: empty", stage)
			continue
		}
		if !strings.EqualFold(filepath.Ext(file), ".ts") {
			addf("config.stages[%s]: must be a .ts file", stage)
		}
	}

	var unknown []string
	for stage := range c.Stages {
		if !slices.Contains(requiredStages, stage) {
			unknown = append(unknown, string(stage))
		}
	}
	slices.Sort(unknown)
	for _, stage := range unknown {
		addf("config.stages[%s]: unknown stage", stage)
	}

	return errors.Join(errs...)
}
