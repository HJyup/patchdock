package config

import (
	"encoding/json"
	"fmt"
	"time"
)

type Duration time.Duration

func (d *Duration) UnmarshalJSON(data []byte) error {
	var value any
	if err := json.Unmarshal(data, &value); err != nil {
		return err
	}

	switch v := value.(type) {
	case string:
		parsed, err := time.ParseDuration(v)
		if err != nil {
			return fmt.Errorf("invalid duration %q: %w", v, err)
		}
		*d = Duration(parsed)
		return nil

	case float64:
		if v == 0 {
			*d = 0
			return nil
		}
	}

	return fmt.Errorf(`duration must be a string like "15m" or 0`)
}

func (d Duration) MarshalJSON() ([]byte, error) {
	return json.Marshal(time.Duration(d).String())
}

func (d Duration) Duration() time.Duration {
	return time.Duration(d)
}
