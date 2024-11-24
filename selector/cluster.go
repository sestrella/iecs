package selector

import (
	"context"
	"fmt"

	"github.com/aws/aws-sdk-go-v2/service/ecs"
	"github.com/aws/aws-sdk-go-v2/service/ecs/types"
)

func SelectCluster(ctx context.Context, client Client, selector Selector, clusterId string) (*types.Cluster, error) {
	if clusterId == "" {
		listClusters, err := client.ListClusters(ctx, &ecs.ListClustersInput{})
		if err != nil {
			return nil, err
		}
		clusterArn, err := selector.Select("Cluster", listClusters.ClusterArns)
		if err != nil {
			return nil, err
		}
		clusterId = clusterArn
	}
	describeClusters, err := client.DescribeClusters(ctx, &ecs.DescribeClustersInput{
		Clusters: []string{clusterId},
	})
	if err != nil {
		return nil, err
	}
	if len(describeClusters.Clusters) > 0 {
		return &describeClusters.Clusters[0], nil
	}
	return nil, fmt.Errorf("no cluster '%v' found", clusterId)
}
