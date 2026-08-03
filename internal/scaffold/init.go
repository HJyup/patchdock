package scaffold

import (
	"embed"
	"errors"
	"fmt"
	"os"
	"path/filepath"
)

//go:embed templates/*
var templates embed.FS

var ErrAlreadyExists = errors.New(".patchdock already exists")

type Options struct {
	RepoDir string
	Force   bool
}

type fileSource struct {
	src string
	dst string
}

var scaffoldFiles = []fileSource{
	{src: "templates/config.template.yml", dst: "config.yml"},
	{src: "templates/planner.ts.tmpl", dst: "planner.ts"},
	{src: "templates/executor.ts.tmpl", dst: "executor.ts"},
	{src: "templates/reviewer.ts.tmpl", dst: "reviewer.ts"},
	{src: "templates/Dockerfile.tmpl", dst: "Dockerfile"},
}

func Init(opts Options) error {
	stats, err := os.Stat(opts.RepoDir)
	if err != nil {
		return fmt.Errorf("failed to get stats for %s: %w", opts.RepoDir, err)
	}
	if !stats.IsDir() {
		return fmt.Errorf("%s is not a directory", opts.RepoDir)
	}

	patchdockDir := filepath.Join(opts.RepoDir, ".patchdock")
	if stats, err := os.Stat(patchdockDir); err == nil {
		if !stats.IsDir() {
			return fmt.Errorf("%s exists and is not a directory", patchdockDir)
		}
		if opts.Force {
			if err := os.RemoveAll(patchdockDir); err != nil {
				return fmt.Errorf("failed to overwrite %s: %w", patchdockDir, err)
			}
		} else {
			return fmt.Errorf("%s: %w", patchdockDir, ErrAlreadyExists)
		}
	} else if !errors.Is(err, os.ErrNotExist) {
		return fmt.Errorf("stat %s: %w", patchdockDir, err)
	}

	if err := os.Mkdir(patchdockDir, 0o755); err != nil {
		return fmt.Errorf("failed to create %s: %w", patchdockDir, err)
	}

	for _, file := range scaffoldFiles {
		if err = writeTemplateFile(patchdockDir, file); err != nil {
			return err
		}
	}

	return nil
}

func writeTemplateFile(folder string, file fileSource) error {
	data, err := templates.ReadFile(file.src)
	if err != nil {
		return fmt.Errorf("read embedded template %s: %w", file.src, err)
	}
	dst := filepath.Join(folder, file.dst)
	if err := os.WriteFile(dst, data, 0o644); err != nil {
		return fmt.Errorf("write %s: %w", dst, err)
	}

	return nil
}
