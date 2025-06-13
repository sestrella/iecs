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

type SelectedContainerDefinition struct {
	Cluster             *types.Cluster
	Service             *types.Service
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
	selection, err := containerDefinitionSelector(ctx, selectors)
	if err != nil {
		return err
	}

	if selection.ContainerDefinition.LogConfiguration == nil {
		return fmt.Errorf(
			"no log configuration found for container: %s",
			*selection.ContainerDefinition.Name,
		)
	}

	logOptions := selection.ContainerDefinition.LogConfiguration.Options
	if len(logOptions) == 0 {
		return fmt.Errorf(
			"missing log options for container: %s",
			*selection.ContainerDefinition.Name,
		)
	}

	awslogsGroup, ok := logOptions["awslogs-group"]
	if !ok {
		return fmt.Errorf(
			"missing awslogs-group option for container: %s",
			*selection.ContainerDefinition.Name,
		)
	}

	streamPrefix, ok := logOptions["awslogs-stream-prefix"]
	if !ok {
		return fmt.Errorf(
			"missing awslogs-stream-prefix option for container: %s",
			*selection.ContainerDefinition.Name,
		)
	}

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

func containerDefinitionSelector(
	ctx context.Context,
	selectors selector.Selectors,
) (*SelectedContainerDefinition, error) {
	cluster, err := selectors.Cluster(ctx)
	if err != nil {
		return nil, err
	}

	service, err := selectors.Service(ctx, cluster)
	if err != nil {
		return nil, err
	}

	containerDefinition, err := selectors.ContainerDefinition(ctx, service)
	if err != nil {
		return nil, err
	}

	return &SelectedContainerDefinition{
		Cluster:             cluster,
		Service:             service,
		ContainerDefinition: containerDefinition,
	}, nil
}

func init() {
	rootCmd.AddCommand(logsCmd)
}
