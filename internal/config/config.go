package config

import "github.com/HJyup/patchdock/internal/types"

type Config struct {
	Container Container                  `yaml:"container"`
	Retries   Retries                    `yaml:"retries"`
	Checks    Check                      `yaml:"checks" validate:"omitempty,min=1,dive"`
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

type Check struct {
	Image string `yaml:"image"`
	Run   []Run  `yaml:"run" validate:"omitempty,dive"`
}

type Run struct {
	Name string `yaml:"name" validate:"required"`
	Cmd  string `yaml:"cmd" validate:"required"`
}
