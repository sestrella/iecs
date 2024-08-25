package cmd

import (
	"fmt"

	"github.com/spf13/cobra"
)

var deployClusterId string

var deployCmd = &cobra.Command{
	Use:   "deploy",
	Short: "A brief description of your command",
	Long:  "TODO",
	Run: func(cmd *cobra.Command, args []string) {
		fmt.Println("deploy called")
	},
}

func init() {
	rootCmd.AddCommand(deployCmd)

	deployCmd.Flags().StringVarP(&deployClusterId, "cluster", "c", "", "TODO")
}
