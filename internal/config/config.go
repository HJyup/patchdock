package config

import "github.com/HJyup/patchdock/internal/types"

type Config struct {
	Container Container                  `yaml:"container"`
	Retries   Retries                    `yaml:"retries"`
	Stages    map[types.StageName]string `yaml:"stages" validate:"dive,keys,oneof=planner executor reviewer,endkeys,required,tsfile"`
}

type Container struct {
	Timeout       Duration `yaml:"timeout" validate:"gte=0"`
	TokenBudget   int      `yaml:"token_budget" validate:"gte=0"`
	MaxContainers int      `yaml:"max_containers" validate:"gte=0"`
}

type Retries struct {
	Max int `yaml:"max" validate:"gte=0"`
}
