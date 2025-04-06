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

func (s *SpyClient) DescribeClusters(
	ctx context.Context,
	input *ecs.DescribeClustersInput,
	options ...func(*ecs.Options),
) (*ecs.DescribeClustersOutput, error) {
	cluster := input.Clusters[0]
	var clusterArn string
	if strings.Contains(cluster, "/") {
		clusterArn = cluster
	} else {
		clusterArn = fmt.Sprintf("arn:aws:ecs:us-east-1:111111111111:cluster/%s", cluster)
	}
	return &ecs.DescribeClustersOutput{Clusters: []types.Cluster{
		{ClusterArn: &clusterArn},
	}}, nil
}

func (s *SpyClient) DescribeServices(
	ctx context.Context,
	input *ecs.DescribeServicesInput,
	options ...func(*ecs.Options),
) (*ecs.DescribeServicesOutput, error) {
	clusterName := strings.Split(*input.Cluster, "/")[1]
	serviceArn := fmt.Sprintf(
		"arn:aws:ecs:us-east-1:111111111111:service/%s/%s",
		clusterName,
		input.Services[0],
	)
	return &ecs.DescribeServicesOutput{Services: []types.Service{
		{ServiceArn: &serviceArn},
	}}, nil
}

func (s *SpyClient) DescribeTasks(
	ctx context.Context,
	input *ecs.DescribeTasksInput,
	options ...func(*ecs.Options),
) (*ecs.DescribeTasksOutput, error) {
	task := input.Tasks[0]
	var taskArn string
	if strings.Contains(task, "arn:aws:ecs") {
		taskArn = task
	} else {
		taskArn = fmt.Sprintf("arn:aws:ecs:us-east-1:111111111111:task/%s", task)
	}
	return &ecs.DescribeTasksOutput{Tasks: []types.Task{
		{TaskArn: &taskArn},
	}}, nil
}

func (s *SpyClient) DescribeTaskDefinition(
	ctx context.Context,
	input *ecs.DescribeTaskDefinitionInput,
	options ...func(*ecs.Options),
) (*ecs.DescribeTaskDefinitionOutput, error) {
	return nil, nil
}

func (s *SpyClient) ListClusters(
	ctx context.Context,
	input *ecs.ListClustersInput,
	options ...func(*ecs.Options),
) (*ecs.ListClustersOutput, error) {
	clusterArns := []string{
		"arn:aws:ecs:us-east-1:111111111111:cluster/cluster-1",
		"arn:aws:ecs:us-east-1:111111111111:cluster/cluster-2",
	}
	return &ecs.ListClustersOutput{ClusterArns: clusterArns}, nil
}

func (s *SpyClient) ListServices(
	ctx context.Context,
	input *ecs.ListServicesInput,
	options ...func(*ecs.Options),
) (*ecs.ListServicesOutput, error) {
	return &ecs.ListServicesOutput{ServiceArns: []string{"service-1", "service-2"}}, nil
}

func (s *SpyClient) ListTasks(
	ctx context.Context,
	input *ecs.ListTasksInput,
	options ...func(*ecs.Options),
) (*ecs.ListTasksOutput, error) {
	clusterSplices := strings.Split(*input.Cluster, "/")
	taskArns := []string{
		fmt.Sprintf(
			"arn:aws:ecs:us-east-1:111111111111:task/%s/7f9ea0d0011a41b7b3f6c37cb29cd25b",
			clusterSplices[1],
		),
		fmt.Sprintf(
			"arn:aws:ecs:us-east-1:111111111111:task/%s/e2c735b1aca94012b37e03d8fe1bfb5f",
			clusterSplices[1],
		),
	}
	return &ecs.ListTasksOutput{TaskArns: taskArns}, nil
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
	clusterArn := "arn:aws:ecs:us-east-1:111111111111:cluster/cluster"

	t.Run("serviceId is empty", func(t *testing.T) {
		service, err := SelectService(context.TODO(), client, selector, clusterArn, "")
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
		service, err := SelectService(context.TODO(), client, selector, clusterArn, "service")
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

func TestSelectorTask(t *testing.T) {
	client := &SpyClient{}
	selector := &SpySelector{}
	clusterArn := "arn:aws:ecs:us-east-1:111111111111:cluster/cluster"
	serviceName := "service"

	t.Run("taskId is empty", func(t *testing.T) {
		task, err := SelectTask(context.TODO(), client, selector, clusterArn, serviceName, "")
		if err != nil {
			t.Fatal(err)
		}
		got := *task.TaskArn
		want := "arn:aws:ecs:us-east-1:111111111111:task/cluster/7f9ea0d0011a41b7b3f6c37cb29cd25b"
		if got != want {
			t.Errorf("got %q want %q", got, want)
		}
	})

	t.Run("taskId is not empty", func(t *testing.T) {
		task, err := SelectTask(
			context.TODO(),
			client,
			selector,
			clusterArn,
			serviceName,
			"cluster/44d9c0c0af0348ec9f57e4e413293c6b",
		)
		if err != nil {
			t.Fatal(err)
		}
		got := *task.TaskArn
		want := "arn:aws:ecs:us-east-1:111111111111:task/cluster/44d9c0c0af0348ec9f57e4e413293c6b"
		if got != want {
			t.Errorf("got %q want %q", got, want)
		}
	})
}
