package cmd

import (
	"context"
	"fmt"
	"time"

	"github.com/aws/aws-sdk-go-v2/config"
	"github.com/sestrella/iecs/client"
	"github.com/sestrella/iecs/selector"
	"github.com/spf13/cobra"
)

var logsCmd = &cobra.Command{
	Use:   "logs",
	Short: "View the logs of a container",
	Example: `
  aws-vault exec <profile> -- iecs logs [flags] (recommended)
  env AWS_PROFILE=<profile> iecs logs [flags]
  `,
	Run: func(cmd *cobra.Command, args []string) {
		cfg, err := config.LoadDefaultConfig(context.TODO())
		if err != nil {
			panic(err)
		}
		awsClient := client.NewClient(cfg)
		err = runLogs(context.TODO(), awsClient)
		if err != nil {
			panic(err)
		}
	},
	Aliases: []string{"tail"},
}

func runLogs(
	ctx context.Context,
	client client.Client,
) error {
	selection, err := selector.RunContainerDefinitionSelector(ctx, client)
	if err != nil {
		return err
	}

	logOptions := selection.ContainerDefinition.LogConfiguration.Options
	awslogsGroup := logOptions["awslogs-group"]
	streamPrefix := logOptions["awslogs-stream-prefix"]

	// Use our logs client to start the live tail
	err = client.StartLiveTail(
		ctx,
		awslogsGroup,
		streamPrefix,
		func(timestamp time.Time, message string) {
			fmt.Printf("%v %s\n", timestamp, message)
		},
	)
	if err != nil {
		return err
	}

	return nil
}

func init() {
	rootCmd.AddCommand(logsCmd)
}
