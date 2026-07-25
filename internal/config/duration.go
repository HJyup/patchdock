package config

import (
	"fmt"
	"time"

	"gopkg.in/yaml.v3"
)

// Duration wraps time.Duration so it can be read from YAML as a human string
// like "10m" or "90s". yaml.v3 cannot do this on time.Duration directly
type Duration time.Duration

func (d *Duration) UnmarshalYAML(value *yaml.Node) error {
	if value.Kind != yaml.ScalarNode {
		return fmt.Errorf("duration must be a string like \"10m\" or 0")
	}

	if value.Value == "0" {
		*d = 0
		return nil
	}

	parsed, err := time.ParseDuration(value.Value)
	if err != nil {
		return fmt.Errorf("invalid duration %q: %w", value.Value, err)
	}

	*d = Duration(parsed)
	return nil
}

func (d Duration) MarshalYAML() (any, error) {
	return time.Duration(d).String(), nil
}

func (d Duration) Duration() time.Duration {
	return time.Duration(d)
}
