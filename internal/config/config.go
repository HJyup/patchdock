package config

type Config struct {
	Container Container         `yaml:"container"`
	Retries   Retries           `yaml:"retries"`
	Checks    []Check           `yaml:"checks" validate:"omitempty,min=1,dive"`
	Stages    map[string]string `yaml:"stages" validate:"dive,keys,oneof=planner executor reviewer,endkeys,required,tsfile"`
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
	Name string `yaml:"name" validate:"required"`
	Run  string `yaml:"run" validate:"required"`
}
