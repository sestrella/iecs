package cmd

import (
	"context"
	"time"

	"github.com/aws/aws-sdk-go-v2/service/ecs/types"
	"github.com/sestrella/iecs/client"
	"github.com/sestrella/iecs/selector"
	"github.com/spf13/cobra"
)

var waitTimeoutFlag time.Duration

type UpdateSelection struct {
	cluster       types.Cluster
	service       types.Service
	serviceConfig client.ServiceConfig
}

var updateCmd = &cobra.Command{
	Use:   "update",
	Short: "Updates a serice configuration",
	RunE: func(cmd *cobra.Command, args []string) error {
		selection, err := updateSelector(
			context.Background(),
			rootSelectors,
		)
		if err != nil {
			return err
		}

		err = runUpdate(context.Background(), *selection, rootClient, waitTimeoutFlag)
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
	serviceConfig, err := selectors.ServiceConfig(ctx, rootService)
	if err != nil {
		return nil, err
	}

	return &UpdateSelection{
		cluster:       *rootCluster,
		service:       *rootService,
		serviceConfig: *serviceConfig,
	}, nil
}

func runUpdate(
	ctx context.Context,
	selection UpdateSelection,
	client client.Client,
	waitTimeout time.Duration,
) error {
	_, err := client.UpdateService(
		ctx,
		&selection.service,
		selection.serviceConfig,
		waitTimeout,
	)
	if err != nil {
		return err
	}

	return nil
}

func init() {
	rootCmd.AddCommand(updateCmd)

	updateCmd.Flags().
		DurationVarP(&waitTimeoutFlag, "wait-timeout", "w", 5*time.Minute, "The wait time for the service to become available")
}
