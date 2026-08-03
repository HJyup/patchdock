package cli

import (
	"fmt"
	"path/filepath"

	"github.com/HJyup/patchdock/internal/app"
	"github.com/spf13/cobra"
)

var force bool

var initCmd = &cobra.Command{
	Use:   "init",
	Short: "Scaffold .patchdock/ in a repository",
	Long: `Writes .patchdock/ with a config, a Dockerfile and three working
			agents, so "dock init" followed by "dock run" works as-is.
			Refuses to touch an existing .patchdock/ unless --force is given.`,
	Args: cobra.MaximumNArgs(1),
	RunE: func(cmd *cobra.Command, args []string) error {
		path, err := app.RunPatchdockInit(force)
		if err != nil {
			return err
		}

		fmt.Fprintf(cmd.OutOrStdout(), "created %s\n", filepath.Join(path, ".patchdock"))
		return nil
	},
}

func init() {
	rootCmd.AddCommand(initCmd)
	initCmd.Flags().BoolVar(&force, "force", false, "overwrite an existing .patchdock/ directory")
}
