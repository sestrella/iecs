package selector

import (
	"context"
	"fmt"

	"github.com/aws/aws-sdk-go-v2/service/ecs/types"
	"github.com/charmbracelet/huh"
	"github.com/sestrella/iecs/client"
)

func ClusterSelector(ctx context.Context, client client.Client) (*types.Cluster, error) {
	clusterArns, err := client.ListClusters(ctx)
	if err != nil {
		return nil, fmt.Errorf("failed to list clusters: %w", err)
	}

	var selectedClusterArn string
	if len(clusterArns) == 1 {
		selectedClusterArn = clusterArns[0]
	} else {
		form := huh.NewForm(
			huh.NewGroup(
				huh.NewSelect[string]().
					Title("Select cluster").
					Options(
						huh.NewOptions(clusterArns...)...,
					).
					Value(&selectedClusterArn).
					WithHeight(5),
			),
		)

		if err = form.Run(); err != nil {
			return nil, err
		}
	}

	cluster, err := client.DescribeCluster(ctx, selectedClusterArn)
	if err != nil {
		return nil, fmt.Errorf("failed to describe cluster after selection: %w", err)
	}

	fmt.Printf("Selected cluster: %s\n", *cluster.ClusterArn)
	return cluster, nil
}
