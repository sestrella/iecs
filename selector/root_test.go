// https://quii.gitbook.io/learn-go-with-tests/go-fundamentals/mocking

package selector

import (
	"context"
	"fmt"
	"strings"
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

func (s *SpyClient) DescribeServices(ctx context.Context, input *ecs.DescribeServicesInput, options ...func(*ecs.Options)) (*ecs.DescribeServicesOutput, error) {
	clusterSplices := strings.Split(*input.Cluster, "/")
	serviceName := input.Services[0]
	serviceArn := fmt.Sprintf("arn:aws:ecs:us-east-1:111111111111:service/%s/%s", clusterSplices[1], serviceName)
	return &ecs.DescribeServicesOutput{Services: []types.Service{
		{ServiceName: &serviceName, ServiceArn: &serviceArn},
	}}, nil
}

func (s *SpyClient) DescribeTasks(ctx context.Context, input *ecs.DescribeTasksInput, options ...func(*ecs.Options)) (*ecs.DescribeTasksOutput, error) {
	return nil, nil
}

func (s *SpyClient) ListClusters(ctx context.Context, input *ecs.ListClustersInput, options ...func(*ecs.Options)) (*ecs.ListClustersOutput, error) {
	return &ecs.ListClustersOutput{ClusterArns: []string{"cluster-1", "cluster-2"}}, nil
}

func (s *SpyClient) ListServices(ctx context.Context, input *ecs.ListServicesInput, options ...func(*ecs.Options)) (*ecs.ListServicesOutput, error) {
	return &ecs.ListServicesOutput{ServiceArns: []string{"service-1", "service-2"}}, nil
}

func (s *SpyClient) ListTasks(ctx context.Context, input *ecs.ListTasksInput, options ...func(*ecs.Options)) (*ecs.ListTasksOutput, error) {
	return nil, nil
}

type SpySelector struct{}

func (s *SpySelector) Select(title string, options []string) (string, error) {
	return options[0], nil
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
		want := "arn:aws:ecs:us-east-1:111111111111:cluster/cluster-1"
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

func TestSelectorService(t *testing.T) {
	client := &SpyClient{}
	selector := &SpySelector{}
	clusterId := "arn:aws:ecs:us-east-1:111111111111:cluster/cluster"

	t.Run("serviceId is empty", func(t *testing.T) {
		service, err := SelectService(context.TODO(), client, selector, clusterId, "")
		if err != nil {
			t.Fatal(err)
		}
		got := *service.ServiceArn
		want := "arn:aws:ecs:us-east-1:111111111111:service/cluster/service-1"
		if got != want {
			t.Errorf("got %q want %q", got, want)
		}
	})

	t.Run("serviceId is not empty", func(t *testing.T) {
		service, err := SelectService(context.TODO(), client, selector, clusterId, "service")
		if err != nil {
			t.Fatal(err)
		}
		got := *service.ServiceArn
		want := "arn:aws:ecs:us-east-1:111111111111:service/cluster/service"
		if got != want {
			t.Errorf("got %q want %q", got, want)
		}
	})
}
