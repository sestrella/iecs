package client

import (
	"context"

	"github.com/aws/aws-sdk-go-v2/service/ecs"
	"github.com/aws/aws-sdk-go-v2/service/ecs/types"
)

type Client interface {
	DescribeClusters(
		ctx context.Context,
		input *ecs.DescribeClustersInput,
		options ...func(*ecs.Options),
	) (*ecs.DescribeClustersOutput, error)
	DescribeServices(
		ctx context.Context,
		input *ecs.DescribeServicesInput,
		options ...func(*ecs.Options),
	) (*ecs.DescribeServicesOutput, error)
	DescribeTasks(
		ctx context.Context,
		input *ecs.DescribeTasksInput,
		options ...func(*ecs.Options),
	) (*ecs.DescribeTasksOutput, error)
	DescribeTaskDefinition(
		ctx context.Context,
		input *ecs.DescribeTaskDefinitionInput,
		options ...func(*ecs.Options),
	) (*ecs.DescribeTaskDefinitionOutput, error)
	ListClusters(
		ctx context.Context,
		input *ecs.ListClustersInput,
		options ...func(*ecs.Options),
	) (*ecs.ListClustersOutput, error)
	ListServices(
		ctx context.Context,
		input *ecs.ListServicesInput,
		options ...func(*ecs.Options),
	) (*ecs.ListServicesOutput, error)
	ListTasks(
		ctx context.Context,
		input *ecs.ListTasksInput,
		options ...func(*ecs.Options),
	) (*ecs.ListTasksOutput, error)
}

type NewClient interface {
	DescribeCluster(ctx context.Context, clusterArn string) (*types.Cluster, error)
	DescribeService(
		ctx context.Context,
		clusterArn string,
		serviceArn string,
	) (*types.Service, error)
	DescribeTask(ctx context.Context, clusterArn string, taskArn string) (*types.Task, error)
	ListClusters(ctx context.Context) ([]string, error)
	ListServices(ctx context.Context, clusterArn string) ([]string, error)
	ListTasks(ctx context.Context, clusterArn string, serviceArn string) ([]string, error)
}

type newClient struct {
	client Client
}

func NewClient(client Client) NewClient {
	return &newClient{
		client: client,
	}
}

func (c *newClient) DescribeCluster(ctx context.Context, clusterArn string) (*types.Cluster, error) {
	output, err := c.client.DescribeClusters(ctx, &ecs.DescribeClustersInput{
		Clusters: []string{clusterArn},
	})
	if err != nil {
		return nil, err
	}
	if len(output.Clusters) == 0 {
		return nil, nil
	}
	return &output.Clusters[0], nil
}

func (c *newClient) DescribeService(ctx context.Context, clusterArn string, serviceArn string) (*types.Service, error) {
	output, err := c.client.DescribeServices(ctx, &ecs.DescribeServicesInput{
		Cluster:  &clusterArn,
		Services: []string{serviceArn},
	})
	if err != nil {
		return nil, err
	}
	if len(output.Services) == 0 {
		return nil, nil
	}
	return &output.Services[0], nil
}

func (c *newClient) DescribeTask(ctx context.Context, clusterArn string, taskArn string) (*types.Task, error) {
	output, err := c.client.DescribeTasks(ctx, &ecs.DescribeTasksInput{
		Cluster: &clusterArn,
		Tasks:   []string{taskArn},
	})
	if err != nil {
		return nil, err
	}
	if len(output.Tasks) == 0 {
		return nil, nil
	}
	return &output.Tasks[0], nil
}

func (c *newClient) ListClusters(ctx context.Context) ([]string, error) {
	output, err := c.client.ListClusters(ctx, &ecs.ListClustersInput{})
	if err != nil {
		return nil, err
	}
	return output.ClusterArns, nil
}

func (c *newClient) ListServices(ctx context.Context, clusterArn string) ([]string, error) {
	output, err := c.client.ListServices(ctx, &ecs.ListServicesInput{
		Cluster: &clusterArn,
	})
	if err != nil {
		return nil, err
	}
	return output.ServiceArns, nil
}

func (c *newClient) ListTasks(ctx context.Context, clusterArn string, serviceArn string) ([]string, error) {
	output, err := c.client.ListTasks(ctx, &ecs.ListTasksInput{
		Cluster:     &clusterArn,
		ServiceName: &serviceArn,
	})
	if err != nil {
		return nil, err
	}
	return output.TaskArns, nil
}
