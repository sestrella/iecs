package cmd

import (
	_ "embed"

	"github.com/spf13/cobra"
)

var rootCmd = &cobra.Command{
	Use:   "iecs",
	Short: "An interactive CLI for ECS",
	Long:  "Performs commons tasks on ECS, such as getting remote access or viewing logs",
}

func Execute(version string) error {
	err := rootCmd.Execute()
	if err != nil {
		return err
	}
	return nil
}
