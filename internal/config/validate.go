package config

import (
	"errors"
	"fmt"
	"path/filepath"
	"regexp"
	"slices"
	"strings"

	"github.com/HJyup/patchdock/internal/types"
)

var requiredStages = []types.StageName{types.StagePlanner, types.StageExecutor, types.StageReviewer}
var namespacePattern = regexp.MustCompile(`^[a-z0-9]+(?:[._-][a-z0-9]+)*$`)

func (c *Config) Validate() error {
	var errs []error
	addf := func(format string, args ...any) {
		errs = append(errs, fmt.Errorf(format, args...))
	}

	if !namespacePattern.MatchString(c.Namespace) {
		addf("config.name_space: must be lowercase letters or digits, separated by '.', '_' or '-'")
	}
	if c.Container.Timeout < 0 {
		addf("config.container.timeout: must be >= 0")
	}
	if c.Container.TokenBudget < 0 {
		addf("config.container.token_budget: must be >= 0")
	}
	if c.Retries.Max < 1 {
		addf("config.retries.max: must be >= 1")
	}
	targets := make(map[string]struct{}, len(c.Credentials))
	for i, cred := range c.Credentials {
		if cred.Host == "" {
			addf("config.credentials[%d].host: missing", i)
		}
		switch {
		case cred.Target == "":
			addf("config.credentials[%d].target: missing", i)
		case !filepath.IsAbs(cred.Target):
			addf("config.credentials[%d].target: must be an absolute container path", i)
		default:
			if _, taken := targets[cred.Target]; taken {
				addf("config.credentials[%d].target: %q is already mounted", i, cred.Target)
			}
			targets[cred.Target] = struct{}{}
		}
		for key := range cred.Env {
			if key == "" {
				addf("config.credentials[%d].env: empty variable name", i)
			}
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
