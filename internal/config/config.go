package config

type Config struct {
	Container Container         `yaml:"container"`
	Retries   Retries           `yaml:"retries"`
	Checks    []Check           `yaml:"checks"`
	Stages    map[string]string `yaml:"stages"`
}

type Container struct {
	Timeout       Duration `yaml:"timeout"`
	TokenBudget   int      `yaml:"token_budget"`
	MaxContainers int      `yaml:"max_containers"`
}

type Retries struct {
	Max int `yaml:"max"`
}

type Check struct {
	Name string `yaml:"name"`
	Run  string `yaml:"run"`
}
