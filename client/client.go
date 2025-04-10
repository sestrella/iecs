package client

import (
	"context"
	"fmt"

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

type ClientV2 interface {
	DescribeCluster(ctx context.Context, clusterArn string) (*types.Cluster, error)
	DescribeService(
		ctx context.Context,
		clusterArn string,
		serviceArn string,
	) (*types.Service, error)
	DescribeTask(ctx context.Context, clusterArn string, taskArn string) (*types.Task, error)
	DescribeTaskDefinition(
		ctx context.Context,
		taskDefinitionArn string,
	) (*types.TaskDefinition, error)
	ListClusters(ctx context.Context) ([]string, error)
	ListServices(ctx context.Context, clusterArn string) ([]string, error)
	ListTasks(ctx context.Context, clusterArn string, serviceArn string) ([]string, error)
}

type newClient struct {
	client ecs.Client
}

func NewClientV2(client ecs.Client) ClientV2 {
	return &newClient{
		client: client,
	}
}

func (c *newClient) DescribeCluster(
	ctx context.Context,
	clusterArn string,
) (*types.Cluster, error) {
	output, err := c.client.DescribeClusters(ctx, &ecs.DescribeClustersInput{
		Clusters: []string{clusterArn},
	})
	if err != nil {
		return nil, err
	}
	if len(output.Clusters) == 0 {
		return nil, fmt.Errorf("cluster not found: %s", clusterArn)
	}
	return &output.Clusters[0], nil
}

func (c *newClient) DescribeService(
	ctx context.Context,
	clusterArn string,
	serviceArn string,
) (*types.Service, error) {
	output, err := c.client.DescribeServices(ctx, &ecs.DescribeServicesInput{
		Cluster:  &clusterArn,
		Services: []string{serviceArn},
	})
	if err != nil {
		return nil, err
	}
	if len(output.Services) == 0 {
		return nil, fmt.Errorf("service not found: %s", serviceArn)
	}
	return &output.Services[0], nil
}

func (c *newClient) DescribeTask(
	ctx context.Context,
	clusterArn string,
	taskArn string,
) (*types.Task, error) {
	output, err := c.client.DescribeTasks(ctx, &ecs.DescribeTasksInput{
		Cluster: &clusterArn,
		Tasks:   []string{taskArn},
	})
	if err != nil {
		return nil, err
	}
	if len(output.Tasks) == 0 {
		return nil, fmt.Errorf("task not found: %s", taskArn)
	}
	return &output.Tasks[0], nil
}

func (c *newClient) DescribeTaskDefinition(
	ctx context.Context,
	taskDefinitionArn string,
) (*types.TaskDefinition, error) {
	output, err := c.client.DescribeTaskDefinition(ctx, &ecs.DescribeTaskDefinitionInput{
		TaskDefinition: &taskDefinitionArn,
	})
	if err != nil {
		return nil, err
	}
	if output.TaskDefinition == nil {
		return nil, fmt.Errorf("task definition not found: %s", taskDefinitionArn)
	}
	return output.TaskDefinition, nil
}

func (c *newClient) ListClusters(ctx context.Context) ([]string, error) {
	output, err := c.client.ListClusters(ctx, &ecs.ListClustersInput{})
	if err != nil {
		return nil, err
	}
	if len(output.ClusterArns) == 0 {
		return []string{}, fmt.Errorf("no clusters found")
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
	if len(output.ServiceArns) == 0 {
		return []string{}, fmt.Errorf("no services found in cluster: %s", clusterArn)
	}
	return output.ServiceArns, nil
}

func (c *newClient) ListTasks(
	ctx context.Context,
	clusterArn string,
	serviceArn string,
) ([]string, error) {
	output, err := c.client.ListTasks(ctx, &ecs.ListTasksInput{
		Cluster:     &clusterArn,
		ServiceName: &serviceArn,
	})
	if err != nil {
		return nil, err
	}
	if len(output.TaskArns) == 0 {
		return []string{}, fmt.Errorf(
			"no tasks found for service: %s in cluster: %s",
			serviceArn,
			clusterArn,
		)
	}
	return output.TaskArns, nil
}
