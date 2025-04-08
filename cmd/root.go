package cmd

import (
	_ "embed"
	"os"

	"github.com/spf13/cobra"
)

//go:embed version.txt
var version string

var rootCmd = &cobra.Command{
	Use:     "iecs",
	Short:   "An interactive CLI for ECS",
	Long:    "Performs commons tasks on ECS, such as getting remote access or viewing logs",
	Version: version,
}

func Execute() {
	err := rootCmd.Execute()
	if err != nil {
		os.Exit(1)
	}
}
