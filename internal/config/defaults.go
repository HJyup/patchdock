package config

import (
	"time"

	"github.com/HJyup/patchdock/internal/utils"
)

const (
	DefaultTimeout      = utils.Duration(10 * time.Minute)
	DefaultTokenBudget  = 100000
	DefaultRetriesMax   = 3
	DefaultBranchPrefix = "patchdock"
)

func Defaults() Config {
	return Config{
		Container: Container{
			Timeout:     DefaultTimeout,
			TokenBudget: DefaultTokenBudget,
		},
		Retries: Retries{
			Max: DefaultRetriesMax,
		},
		Git: Git{
			BranchPrefix: DefaultBranchPrefix,
		},
	}
}
