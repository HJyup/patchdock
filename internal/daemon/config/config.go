package config

import (
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"os"
	"time"

	"github.com/HJyup/patchdock/internal/utils"
)

const (
	DefaultMaxContainers = 3
	DefaultRetention     = utils.Duration(15 * time.Minute)
	DefaultSnapshotTick  = utils.Duration(200 * time.Millisecond)
	DefaultInboxSize     = 256
)

type Config struct {
	MaxContainers int            `json:"max_containers"`
	Retention     utils.Duration `json:"retention"`
	SnapshotTick  utils.Duration `json:"snapshot_tick"`
	InboxSize     int            `json:"inbox_size"`
}

func Defaults() Config {
	return Config{
		MaxContainers: DefaultMaxContainers,
		Retention:     DefaultRetention,
		SnapshotTick:  DefaultSnapshotTick,
		InboxSize:     DefaultInboxSize,
	}
}

func Load(path string) (*Config, error) {
	file, err := os.Open(path)
	if errors.Is(err, os.ErrNotExist) {
		return create(path)
	}
	if err != nil {
		return nil, fmt.Errorf("open config %s: %w", path, err)
	}
	defer file.Close()

	cfg := Defaults()
	decoder := json.NewDecoder(file)
	decoder.DisallowUnknownFields()

	if err := decoder.Decode(&cfg); err != nil {
		if errors.Is(err, io.EOF) {
			return nil, fmt.Errorf("config %s is empty: delete it to recreate it with defaults", path)
		}
		return nil, fmt.Errorf("decode config %s: %w", path, err)
	}

	if err := cfg.Validate(); err != nil {
		return nil, fmt.Errorf("validate config %s: %w", path, err)
	}

	return &cfg, nil
}

func create(path string) (*Config, error) {
	cfg := Defaults()

	encoded, err := json.MarshalIndent(cfg, "", "  ")
	if err != nil {
		return nil, fmt.Errorf("encode config %s: %w", path, err)
	}
	encoded = append(encoded, '\n')

	if err := os.WriteFile(path, encoded, 0o600); err != nil {
		return nil, fmt.Errorf("write config %s: %w", path, err)
	}

	return &cfg, nil
}

func (c *Config) Validate() error {
	var errs []error

	if c.MaxContainers < 1 {
		errs = append(errs, errors.New("config.max_containers: must be >= 1"))
	}
	if c.Retention <= 0 {
		errs = append(errs, errors.New("config.retention: must be > 0"))
	}
	if c.InboxSize <= 0 {
		errs = append(errs, errors.New("config.inbox_size: must be >= 1"))
	}
	if c.SnapshotTick <= 0 {
		errs = append(errs, errors.New("config.snapshot_tick: must be > 0"))
	}

	return errors.Join(errs...)
}
