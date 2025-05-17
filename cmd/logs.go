package cmd

import (
	"context"
	"fmt"
	"time"

	"github.com/aws/aws-sdk-go-v2/config"
	"github.com/aws/aws-sdk-go-v2/service/ecs"
	"github.com/aws/aws-sdk-go-v2/service/ecs/types"
	"github.com/sestrella/iecs/client"
	"github.com/sestrella/iecs/selector"
	"github.com/spf13/cobra"
)

type selectedContainerDefinition struct {
	Cluster             *types.Cluster
	Service             *types.Service
	TaskDefinition      *types.TaskDefinition
	ContainerDefinition *types.ContainerDefinition
}

var logsCmd = &cobra.Command{
	Use:   "logs",
	Short: "View the logs of a container",
	Example: `
  aws-vault exec <profile> -- iecs logs [flags] (recommended)
  env AWS_PROFILE=<profile> iecs logs [flags]
  `,
	RunE: func(cmd *cobra.Command, args []string) error {
		cfg, err := config.LoadDefaultConfig(context.TODO())
		if err != nil {
			return err
		}
		client := client.NewClient(cfg)
		ecsClient := ecs.NewFromConfig(cfg)
		err = runLogs(context.TODO(), client, selector.NewSelectors(ecsClient))
		if err != nil {
			return err
		}
		return nil
	},
	Aliases: []string{"tail"},
}

func runLogs(
	ctx context.Context,
	client client.Client,
	selectors selector.Selectors,
) error {
	selection, err := selectors.RunContainerDefinitionSelector(ctx)
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

// runContainerDefinitionSelector runs an interactive form to select an ECS cluster, service and container definition
// Keep for backward compatibility, uses the new interface method
func runContainerDefinitionSelector(
	ctx context.Context,
	selectors selector.Selectors,
) (*selectedContainerDefinition, error) {
	selection, err := selectors.RunContainerDefinitionSelector(ctx)
	if err != nil {
		return nil, err
	}
	return &selectedContainerDefinition{
		Cluster:             selection.Cluster,
		Service:             selection.Service,
		ContainerDefinition: selection.ContainerDefinition,
	}, nil
}

func init() {
	rootCmd.AddCommand(logsCmd)
}
