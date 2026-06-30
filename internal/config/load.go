package config

import (
	"fmt"
	"os"

	"gopkg.in/yaml.v3"
)

func Load(filePath string) (Config, error) {
	file, err := os.Open(filePath)
	if err != nil {
		return Config{}, fmt.Errorf("read config %s: %w", filePath, err)
	}
	defer file.Close()

	cfg := Defaults()
	decoder := yaml.NewDecoder(file)
	decoder.KnownFields(true)
	if err := decoder.Decode(&cfg); err != nil {
		return Config{}, fmt.Errorf("parse config %s: %w", filePath, err)
	}

	if err := cfg.Validate(); err != nil {
		return Config{}, fmt.Errorf("validate config %s: %w", filePath, err)
	}

	return cfg, nil
}
