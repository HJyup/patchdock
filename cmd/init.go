package cmd

import (
	"fmt"
	"path/filepath"

	"github.com/HJyup/patchdock/internal/scaffold"
	"github.com/spf13/cobra"
)

var initForce bool

var initCmd = &cobra.Command{
	Use:   "init [repo-dir]",
	Short: "Scaffold .patchdock/ in the current repository",
	Long: `Creates the .patchdock/ directory with everything a repo needs.
		The generated agents work out of the box, so "patchdock init" followed
		by "patchdock run" succeeds before you have written a single line.
		If .patchdock/ already exists the command refuses to touch it; pass
		--force to overwrite the existing files.`,
	Args: cobra.MaximumNArgs(1),
	RunE: func(cmd *cobra.Command, args []string) error {
		repoDir, err := resolveDir(args)
		if err != nil {
			return err
		}

		if err := scaffold.Init(scaffold.Options{RepoDir: repoDir, Force: initForce}); err != nil {
			return err
		}

		fmt.Fprintf(cmd.OutOrStdout(), "created %s\n", filepath.Join(repoDir, ".patchdock"))
		return nil
	},
}

func init() {
	rootCmd.AddCommand(initCmd)

	initCmd.Flags().BoolVar(&initForce, "force", false, "overwrite an existing .patchdock/ directory")
}

func resolveDir(args []string) (string, error) {
	repoDir := "."
	if len(args) > 0 {
		repoDir = args[0]
	}

	abs, err := filepath.Abs(repoDir)
	if err != nil {
		return "", fmt.Errorf("resolve repo dir %s: %w", repoDir, err)
	}
	return abs, nil
}
