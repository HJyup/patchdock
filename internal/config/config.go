package config

import "github.com/HJyup/patchdock/internal/types"

type Config struct {
	Namespace   string                     `yaml:"name_space"`
	Container   Container                  `yaml:"container"`
	Retries     Retries                    `yaml:"retries"`
	Credentials []Credential               `yaml:"credentials,omitempty"`
	Git         Git                        `yaml:"git"`
	Stages      map[types.StageName]string `yaml:"stages"`
}

type Git struct {
	BranchPrefix string `yaml:"branch_prefix"`
}

type Credential struct {
	Host   string            `yaml:"host"`
	Target string            `yaml:"target"`
	Env    map[string]string `yaml:"env,omitempty"`
}

type Container struct {
	Timeout     Duration `yaml:"timeout"`
	TokenBudget int      `yaml:"token_budget"`
}

type Retries struct {
	Max int `yaml:"max"`
}

// imageTagPrefix names every agent image patchdock builds
const imageTagPrefix = "patchdock-agent"

// ImageTag is the agent image this config builds and runs. It is derived from
// the namespace so two repos never collide on one image
func (c Config) ImageTag() string {
	return imageTagPrefix + "-" + c.Namespace
}
