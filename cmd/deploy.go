package cmd

import (
	"context"
	"fmt"
	"log"

	"github.com/aws/aws-sdk-go-v2/config"
	"github.com/aws/aws-sdk-go-v2/service/ecs"
	"github.com/spf13/cobra"
)

var deployClusterId string

var deployCmd = &cobra.Command{
	Use:   "deploy",
	Short: "A brief description of your command",
	Long:  "TODO",
	Run: func(cmd *cobra.Command, args []string) {
		cfg, err := config.LoadDefaultConfig(context.TODO())
		if err != nil {
			log.Fatal(err)
		}

		client := ecs.NewFromConfig(cfg)
		cluster, err := selectCluster(context.TODO(), client, sshClusterId)
		if err != nil {
			log.Fatal(err)
		}
		fmt.Println(cluster)
	},
}

func init() {
	rootCmd.AddCommand(deployCmd)

	deployCmd.Flags().StringVarP(&deployClusterId, "cluster", "c", "", "TODO")
}
