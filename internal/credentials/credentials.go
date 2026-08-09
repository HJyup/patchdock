package credentials

import (
	"fmt"
	"os"
	"path/filepath"
	"strings"

	"github.com/HJyup/patchdock/internal/config"
	"github.com/HJyup/patchdock/internal/docker"
)

// Resolve turns the credentials declared in config.yml into mounts and
// environment shared by every stage container. Host paths are resolved and
// checked here so a missing credential fails before the first container starts.
func Resolve(creds []config.Credential) ([]docker.Mount, map[string]string, error) {
	mounts := make([]docker.Mount, 0, len(creds))
	env := make(map[string]string)

	for _, cred := range creds {
		source, err := hostPath(cred.Host)
		if err != nil {
			return nil, nil, err
		}
		if _, err := os.Stat(source); err != nil {
			return nil, nil, fmt.Errorf("find credential %s: %w", source, err)
		}

		mounts = append(mounts, docker.Mount{Source: source, Target: cred.Target, ReadOnly: true})
		for key, value := range cred.Env {
			if _, taken := env[key]; taken {
				return nil, nil, fmt.Errorf("credential environment variable %q is declared twice", key)
			}
			env[key] = value
		}
	}

	return mounts, env, nil
}

func hostPath(path string) (string, error) {
	if path == "~" || strings.HasPrefix(path, "~/") {
		home, err := os.UserHomeDir()
		if err != nil {
			return "", fmt.Errorf("resolve user home for %s: %w", path, err)
		}
		path = filepath.Join(home, strings.TrimPrefix(path, "~"))
	}

	absolute, err := filepath.Abs(path)
	if err != nil {
		return "", fmt.Errorf("resolve credential path %s: %w", path, err)
	}
	return absolute, nil
}
