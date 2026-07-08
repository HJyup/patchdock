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

type templateFile struct {
	src string
	dst string
}

var scaffoldFiles = []templateFile{
	{src: "templates/config.template.yml", dst: "config.yml"},
	{src: "templates/planner.ts.tmpl", dst: "planner.ts"},
	{src: "templates/executor.ts.tmpl", dst: "executor.ts"},
	{src: "templates/reviewer.ts.tmpl", dst: "reviewer.ts"},
}

func Init(opts Options) error {
	repoDir := opts.RepoDir
	if repoDir == "" {
		cwd, err := os.Getwd()
		if err != nil {
			return fmt.Errorf("resolve current directory: %w", err)
		}
		repoDir = cwd
	}

	stats, err := os.Stat(repoDir)
	if err != nil {
		return fmt.Errorf("stat repo dir %s: %w", repoDir, err)
	}
	if !stats.IsDir() {
		return fmt.Errorf("repo dir %s is not a directory", repoDir)
	}

	phdDir := filepath.Join(repoDir, ".patchdock")
	if stats, err := os.Stat(phdDir); err == nil {
		if !stats.IsDir() {
			return fmt.Errorf("%s exists and is not a directory", phdDir)
		}
		if opts.Force {
			if err := os.RemoveAll(phdDir); err != nil {
				return fmt.Errorf("overwrite %s: %w", phdDir, err)
			}
		} else {
			return fmt.Errorf("%s: %w", phdDir, ErrAlreadyExists)
		}
	} else if !errors.Is(err, os.ErrNotExist) {
		return fmt.Errorf("stat %s: %w", phdDir, err)
	}

	if err := os.Mkdir(phdDir, 0o755); err != nil {
		return fmt.Errorf("create %s: %w", phdDir, err)
	}

	for _, file := range scaffoldFiles {
		if err = processFile(phdDir, file); err != nil {
			return err
		}
	}

	return nil
}

func processFile(folder string, file templateFile) error {
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
