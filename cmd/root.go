package cmd

import (
	"os"

	"github.com/spf13/cobra"
)

var rootCmd = &cobra.Command{
	Use:     "iecs",
	Short:   "An interactive CLI for ECS",
	Version: "0.1.0",
	Example: `
  aws-vault exec <profile> -- iecs ... (recommended)
  env AWS_PROFILE=<profile> iecs ...
  `,
}

func Execute() {
	err := rootCmd.Execute()
	if err != nil {
		os.Exit(1)
	}
}

func init() {
	rootCmd.SetVersionTemplate("{{.Version}}")
}
