package cmd

import (
	"os"
	"github.com/spf13/cobra"
)

var detachMode bool

var rootCmd = &cobra.Command{
	Use: "mini-docker",
}

func Execute() {
	runCmd.Flags().BoolVarP(&detachMode, "detach", "d", false, "Run container in background")
	rootCmd.AddCommand(runCmd)
	rootCmd.AddCommand(childCmd)
	if err := rootCmd.Execute(); err != nil {
		os.Exit(1)
	}
}