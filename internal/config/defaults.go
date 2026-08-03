package config

import "time"

const (
	DefaultTimeout       = Duration(10 * time.Minute)
	DefaultTokenBudget   = 100000
	DefaultMaxContainers = 4
	DefaultRetriesMax    = 3
	DefaultBranchPrefix  = "patchdock"
)

func Defaults() Config {
	return Config{
		Container: Container{
			Timeout:       DefaultTimeout,
			TokenBudget:   DefaultTokenBudget,
			MaxContainers: DefaultMaxContainers,
		},
		Retries: Retries{
			Max: DefaultRetriesMax,
		},
		Git: Git{
			BranchPrefix: DefaultBranchPrefix,
		},
	}
}
