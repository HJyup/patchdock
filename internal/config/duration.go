package config

import (
	"fmt"
	"time"

	"gopkg.in/yaml.v3"
)

// Duration wraps time.Duration so it can be read from YAML as a human string
// like "10m" or "90s". yaml.v3 cannot do this on time.Duration directly
type Duration time.Duration

// UnmarshalYAML parses a Go duration string (time.ParseDuration) into Duration
// Signature of yaml.Unmarshaler, and yaml.v3 only calls the method if it matches
func (d *Duration) UnmarshalYAML(value *yaml.Node) error {
	if value.Kind != yaml.ScalarNode {
		return fmt.Errorf("duration must be a string like \"10m\" or 0")
	}

	if value.Tag == "!!int" {
		var n int64
		if err := value.Decode(&n); err != nil {
			return fmt.Errorf("duration must be a string like \"10m\" or 0: %w", err)
		}
		if n != 0 {
			return fmt.Errorf("numeric duration must be 0 for unlimited; use a string like \"10m\" for bounded timeouts")
		}
		*d = 0
		return nil
	}

	var s string
	if err := value.Decode(&s); err != nil {
		return fmt.Errorf("duration must be a string like \"10m\" or 0: %w", err)
	}
	if s == "0" {
		*d = 0
		return nil
	}

	parsed, err := time.ParseDuration(s)
	if err != nil {
		return fmt.Errorf("invalid duration %q: %w", s, err)
	}

	*d = Duration(parsed)
	return nil
}

// MarshalYAML renders the duration back to its string form
func (d Duration) MarshalYAML() (any, error) {
	return time.Duration(d).String(), nil
}

// Duration returns the underlying time.Duration for use in Go code
func (d Duration) Duration() time.Duration {
	return time.Duration(d)
}
