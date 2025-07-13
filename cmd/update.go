package cmd

import (
	"context"

	"github.com/aws/aws-sdk-go-v2/config"
	"github.com/aws/aws-sdk-go-v2/service/ecs/types"
	"github.com/sestrella/iecs/client"
	"github.com/sestrella/iecs/selector"
	"github.com/spf13/cobra"
)

type UpdateSelection struct {
	cluster        types.Cluster
	service        types.Service
	taskDefinition types.TaskDefinition
}

var updateCmd = &cobra.Command{
	Use:   "update",
	Short: "TODO",
	RunE: func(cmd *cobra.Command, args []string) error {
		cfg, err := config.LoadDefaultConfig(context.Background())
		if err != nil {
			return err
		}

		client := client.NewClient(cfg)
		selectors := selector.NewSelectors(client, *theme)
		selection, err := updateSelector(context.Background(), selectors)
		if err != nil {
			return err
		}

		err = runUpdate(context.Background(), *selection, client)
		if err != nil {
			return err
		}

		return nil
	},
}

func updateSelector(
	ctx context.Context,
	selectors selector.Selectors,
) (*UpdateSelection, error) {
	cluster, err := selectors.Cluster(ctx)
	if err != nil {
		return nil, err
	}

	service, err := selectors.Service(ctx, cluster)
	if err != nil {
		return nil, err
	}

	taskDefinition, err := selectors.TaskDefinition(ctx, *service.TaskDefinition)
	if err != nil {
		return nil, err
	}

	return &UpdateSelection{
		cluster:        *cluster,
		service:        *service,
		taskDefinition: *taskDefinition,
	}, nil
}

func runUpdate(
	ctx context.Context,
	selection UpdateSelection,
	client client.Client,
) error {
	return nil
}

func init() {
	rootCmd.AddCommand(updateCmd)
}
