package cmd

import (
	_ "embed"
	"io"
	"log"

	"github.com/spf13/cobra"
)

var silent bool

var rootCmd = &cobra.Command{
	Use:   "iecs",
	Short: "An interactive CLI for ECS",
	Long:  "Performs commons tasks on ECS, such as getting remote access or viewing logs",
}

func init() {
	rootCmd.PersistentFlags().BoolVar(&silent, "silent", false, "Do not print logs")
}

func Execute(version string) error {
	rootCmd.Version = version

	if silent {
		log.SetOutput(io.Discard)
	}

	err := rootCmd.Execute()
	if err != nil {
		return err
	}
	return nil
}
