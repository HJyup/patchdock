package config

import (
	"time"

	"github.com/HJyup/patchdock/internal/id"
)

const (
	DefaultTimeout       = Duration(10 * time.Minute)
	DefaultTokenBudget   = 100000
	DefaultMaxContainers = 4
	DefaultRetriesMax    = 1
)

func Defaults() Config {
	id := id.New("")

	return Config{
		ID: id,
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
