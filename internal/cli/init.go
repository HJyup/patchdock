package cli

import (
	"errors"
	"fmt"
	"os"
	"path/filepath"

	"github.com/HJyup/patchdock/internal/scaffold"
	"github.com/spf13/cobra"
)

var force bool

var initCmd = &cobra.Command{
	Use:   "init",
	Short: "Scaffold .patchdock/ in a repository",
	Long: `Writes .patchdock/ with a config, a Dockerfile, a package.json pinning the
			agent SDK, and three working agents, so "dock init", "npm install" and
			"dock run" work as-is.
			Refuses to touch an existing .patchdock/ unless --force is given.`,
	Args: cobra.MaximumNArgs(1),
	RunE: func(cmd *cobra.Command, args []string) error {
		repoDir, err := os.Getwd()
		if err != nil {
			return fmt.Errorf("resolve repo dir %s: %w", repoDir, err)
		}

		if err := scaffold.Init(scaffold.Options{RepoDir: repoDir, Force: force}); err != nil {
			if errors.Is(err, scaffold.ErrAlreadyExists) {
				return fmt.Errorf("%s already has .patchdock. Rerun with --force to regenerate it (overwrites config.yml and your agent files)", repoDir)
			}
			return err
		}

		dir := filepath.Join(repoDir, ".patchdock")
		fmt.Fprintf(cmd.OutOrStdout(), "created %s\n\n", dir)

		fmt.Fprintf(cmd.OutOrStdout(), "next: install the agent SDK\n\n  cd %s && npm install\n", dir)
		return nil
	},
}

func init() {
	rootCmd.AddCommand(initCmd)
	initCmd.Flags().BoolVar(&force, "force", false, "overwrite an existing .patchdock/ directory")
}
