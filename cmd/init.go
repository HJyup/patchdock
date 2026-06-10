package cmd

import (
	"fmt"

	"github.com/spf13/cobra"
)

var initForce bool

var initCmd = &cobra.Command{
	Use:   "init",
	Short: "Scaffold .patchdock/ in the current repository",
	Long: `Creates the .patchdock/ directory with everything a repo needs. 
		The generated agents work out of the box, so "patchdock init" followed
		by "patchdock run" succeeds before you have written a single line.
		If .patchdock/ already exists the command refuses to touch it; pass
		--force to overwrite the existing files.`,
	Run: func(cmd *cobra.Command, args []string) {
		if initForce {
			fmt.Println("patchdock init: (skeleton) would overwrite the existing .patchdock/ (--force)")
			return
		}
		fmt.Println("patchdock init: (skeleton) would scaffold .patchdock/ (refusing if it already exists; use --force to overwrite)")
	},
}

func init() {
	rootCmd.AddCommand(initCmd)

	initCmd.Flags().BoolVar(&initForce, "force", false, "overwrite an existing .patchdock/ directory")
}
