package config

import "github.com/HJyup/patchdock/internal/types"

type CodexAuth string

const (
	CodexHostLogin CodexAuth = "host-login"
)

type Config struct {
	Container Container                  `yaml:"container"`
	Retries   Retries                    `yaml:"retries"`
	Codex     *CodexConfig               `yaml:"codex,omitempty"`
	Stages    map[types.StageName]string `yaml:"stages"`
}

type CodexConfig struct {
	Auth CodexAuth `yaml:"auth"`
}

type Container struct {
	Timeout       Duration `yaml:"timeout"`
	TokenBudget   int      `yaml:"token_budget"`
	MaxContainers int      `yaml:"max_containers"`
}

type Retries struct {
	Max int `yaml:"max"`
}
