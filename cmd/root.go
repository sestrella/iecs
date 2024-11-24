package cmd

import (
	"os"

	"github.com/spf13/cobra"
)

const (
	CLUSTER_FLAG   = "cluster"
	SERVICE_FLAG   = "service"
	TASK_FLAG      = "task"
	CONTAINER_FLAG = "container"
)

var rootCmd = &cobra.Command{
	Use:     "iecs",
	Short:   "An interactive CLI for ECS",
	Long:    "Performs commons tasks on ECS, such as getting remote access or viewing logs",
	Version: "0.1.0",
}

func Execute() {
	err := rootCmd.Execute()
	if err != nil {
		os.Exit(1)
	}
}

func init() {
	rootCmd.PersistentFlags().StringP(CLUSTER_FLAG, "l", "", "cluster id or ARN")
	rootCmd.PersistentFlags().StringP(SERVICE_FLAG, "s", "", "service id or ARN")
	rootCmd.PersistentFlags().StringP(TASK_FLAG, "t", "", "task id or ARN")
	rootCmd.PersistentFlags().StringP(CONTAINER_FLAG, "n", "", "container name")
}
