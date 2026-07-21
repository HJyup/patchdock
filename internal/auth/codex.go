package auth

import (
	"fmt"
	"os"
	"path/filepath"

	"github.com/HJyup/patchdock/internal/config"
	"github.com/HJyup/patchdock/internal/docker"
)

const CodexHome = "/codex-auth"

// Credentials contains everything the stage container needs to use Codex.
type Credentials struct {
	Env    map[string]string
	Mounts []docker.Mount
}

func LoadCodex(cfg *config.CodexConfig) (Credentials, error) {
	if cfg == nil {
		return Credentials{}, nil
	}

	switch cfg.Auth {
	case config.CodexHostLogin:
		mount, err := loadCodexAuth()
		if err != nil {
			return Credentials{}, err
		}
		return Credentials{
			Env:    map[string]string{"CODEX_HOME": CodexHome},
			Mounts: []docker.Mount{mount},
		}, nil
	case "":
		return Credentials{}, fmt.Errorf("Codex auth is missing")
	default:
		return Credentials{}, fmt.Errorf("unsupported Codex auth %q", cfg.Auth)
	}
}

func loadCodexAuth() (docker.Mount, error) {
	home, err := os.UserHomeDir()
	if err != nil {
		return docker.Mount{}, fmt.Errorf("resolve user home: %w", err)
	}

	authFile := filepath.Join(home, ".codex", "auth.json")
	if _, err := os.Stat(authFile); err != nil {
		return docker.Mount{}, fmt.Errorf("find Codex credentials at %s: %w", authFile, err)
	}

	return docker.Mount{
		Source:   authFile,
		Target:   CodexHome + "/auth.json",
		ReadOnly: true,
	}, nil
}
