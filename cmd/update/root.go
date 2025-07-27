package update

import (
	"context"
	"time"

	"github.com/aws/aws-sdk-go-v2/config"
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

var Cmd = &cobra.Command{
	Use:   "update",
	Short: "Updates a serice configuration",
	RunE: func(cmd *cobra.Command, args []string) error {
		cfg, err := config.LoadDefaultConfig(context.Background())
		if err != nil {
			return err
		}

		client := client.NewClient(cfg)
		selectors := selector.NewSelectors(client, cmd.Flag("theme").Value.String())
		selection, err := updateSelector(context.Background(), selectors)
		if err != nil {
			return err
		}

		err = runUpdate(context.Background(), *selection, client, waitTimeoutFlag)
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

	serviceConfig, err := selectors.ServiceConfig(ctx, service)
	if err != nil {
		return nil, err
	}

	return &UpdateSelection{
		cluster:       *cluster,
		service:       *service,
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
	Cmd.Flags().
		DurationVarP(&waitTimeoutFlag, "wait-timeout", "w", 5*time.Minute, "The wait time for the service to become available")
}
