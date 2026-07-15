package config

import "github.com/HJyup/patchdock/internal/types"

type Config struct {
	Container Container                  `yaml:"container"`
	Retries   Retries                    `yaml:"retries"`
	Stages    map[types.StageName]string `yaml:"stages"`
}

type Container struct {
	Timeout       Duration `yaml:"timeout"`
	TokenBudget   int      `yaml:"token_budget"`
	MaxContainers int      `yaml:"max_containers"`
}

type Retries struct {
	Max int `yaml:"max"`
}
