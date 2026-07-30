package app

import (
	"errors"
	"fmt"
	"path/filepath"

	"github.com/HJyup/patchdock/internal/scaffold"
)

func RunPatchdockInit(initForce bool) (string, error) {
	repoDir, err := filepath.Abs(".")
	if err != nil {
		return "", fmt.Errorf("resolve repo dir %s: %w", repoDir, err)
	}

	if err := scaffold.Init(scaffold.Options{RepoDir: repoDir, Force: initForce}); err != nil {
		if errors.Is(err, scaffold.ErrAlreadyExists) {
			return "", fmt.Errorf("%s already has .patchdock. Rerun with --force to regenerate it (overwrites config.yml and your agent files)", repoDir)
		}
		return "", err
	}

	return repoDir, nil
}
