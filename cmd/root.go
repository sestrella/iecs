package cmd

import (
	"context"
	"errors"
	"fmt"
	"os"

	"github.com/aws/aws-sdk-go-v2/service/ecs"
	"github.com/aws/aws-sdk-go-v2/service/ecs/types"
	"github.com/pterm/pterm"
	"github.com/spf13/cobra"
)

var rootCmd = &cobra.Command{
	Use:     "interactive-ecs",
	Short:   "An interactive CLI for ECS",
	Version: "0.1.0",
	Example: `
    aws-vault exec <profile> -- interactive-ecs ... (recommended)
    env AWS_PROFILE=<profile> interactive-ecs ...
  `,
}

func selectCluster(ctx context.Context, client *ecs.Client, clusterId string) (*types.Cluster, error) {
	if clusterId == "" {
		clusters, err := client.ListClusters(ctx, &ecs.ListClustersInput{})
		if err != nil {
			return nil, fmt.Errorf("Error listing clusters: %w", err)
		}
		if len(clusters.ClusterArns) == 0 {
			return nil, errors.New("No clusters found")
		}
		clusterArn, err := pterm.DefaultInteractiveSelect.WithOptions(clusters.ClusterArns).Show("Select a cluster")
		if err != nil {
			return nil, fmt.Errorf("Error selecting a cluster: %w", err)
		}
		return describeCluster(ctx, client, clusterArn)
	}
	return describeCluster(ctx, client, clusterId)
}

func describeCluster(ctx context.Context, client *ecs.Client, clusterId string) (*types.Cluster, error) {
	clusters, err := client.DescribeClusters(ctx, &ecs.DescribeClustersInput{
		Clusters: []string{clusterId},
	})
	if err != nil {
		return nil, fmt.Errorf("Error describing cluster '%s': %w", clusterId, err)
	}
	if len(clusters.Clusters) == 0 {
		return nil, fmt.Errorf("No cluster '%s' found", clusterId)
	}
	cluster := clusters.Clusters[0]
	return &cluster, nil
}

func Execute() {
	err := rootCmd.Execute()
	if err != nil {
		os.Exit(1)
	}
}

func init() {
	rootCmd.SetVersionTemplate("{{.Version}}")
}
