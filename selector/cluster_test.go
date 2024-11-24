package selector

import (
	"context"
	"fmt"
	"testing"

	"github.com/aws/aws-sdk-go-v2/service/ecs"
	"github.com/aws/aws-sdk-go-v2/service/ecs/types"
)

type SpyClient struct{}

func (s *SpyClient) DescribeClusters(ctx context.Context, input *ecs.DescribeClustersInput, options ...func(*ecs.Options)) (*ecs.DescribeClustersOutput, error) {
	clusterName := input.Clusters[0]
	clusterArn := fmt.Sprintf("arn:aws:ecs:us-east-1:111111111111:cluster/%s", clusterName)
	return &ecs.DescribeClustersOutput{Clusters: []types.Cluster{
		{ClusterName: &clusterName, ClusterArn: &clusterArn},
	}}, nil
}

func (s *SpyClient) ListClusters(ctx context.Context, input *ecs.ListClustersInput, options ...func(*ecs.Options)) (*ecs.ListClustersOutput, error) {
	return &ecs.ListClustersOutput{ClusterArns: []string{}}, nil
}

type SpySelector struct{}

func (s *SpySelector) Select(title string, options []string) (string, error) {
	return "selected-cluster", nil
}

func TestSelectorCluster(t *testing.T) {
	client := &SpyClient{}
	selector := &SpySelector{}

	t.Run("clusterId is empty", func(t *testing.T) {
		cluster, err := SelectCluster(context.TODO(), client, selector, "")
		if err != nil {
			t.Fatal(err)
		}
		got := *cluster.ClusterArn
		want := "arn:aws:ecs:us-east-1:111111111111:cluster/selected-cluster"
		if got != want {
			t.Errorf("got %q want %q", got, want)
		}
	})

	t.Run("clusterId is not empty", func(t *testing.T) {
		cluster, err := SelectCluster(context.TODO(), client, selector, "cluster")
		if err != nil {
			t.Fatal(err)
		}
		got := *cluster.ClusterArn
		want := "arn:aws:ecs:us-east-1:111111111111:cluster/cluster"
		if got != want {
			t.Errorf("got %q want %q", got, want)
		}
	})
}
