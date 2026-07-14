package config

import "time"

const (
	DefaultTimeout       = Duration(10 * time.Minute)
	DefaultTokenBudget   = 100000
	DefaultMaxContainers = 4
	DefaultRetriesMax    = 1
	DefaultImage         = "node:22-slim"
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
	}
}
