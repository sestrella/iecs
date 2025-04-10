package client

import (
	"context"
	"fmt"

	"github.com/aws/aws-sdk-go-v2/service/ecs"
	"github.com/aws/aws-sdk-go-v2/service/ecs/types"
)

type EcsClient interface {
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
	ExecuteCommand(
		ctx context.Context,
		clusterArn *string,
		taskArn *string,
		containerName *string,
		command string,
		interactive bool,
	) (*ecs.ExecuteCommandOutput, error)
}

type awsEcsClient struct {
	client *ecs.Client
}

func NewEcsClient(client *ecs.Client) EcsClient {
	return awsEcsClient{
		client: client,
	}
}

func (c awsEcsClient) DescribeCluster(
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

func (c awsEcsClient) DescribeService(
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

func (c awsEcsClient) DescribeTask(
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

func (c awsEcsClient) DescribeTaskDefinition(
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

func (c awsEcsClient) ListClusters(ctx context.Context) ([]string, error) {
	output, err := c.client.ListClusters(ctx, &ecs.ListClustersInput{})
	if err != nil {
		return nil, err
	}
	if len(output.ClusterArns) == 0 {
		return []string{}, fmt.Errorf("no clusters found")
	}
	return output.ClusterArns, nil
}

func (c awsEcsClient) ListServices(ctx context.Context, clusterArn string) ([]string, error) {
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

func (c awsEcsClient) ListTasks(
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

func (c awsEcsClient) ExecuteCommand(
	ctx context.Context,
	clusterArn *string,
	taskArn *string,
	containerName *string,
	command string,
	interactive bool,
) (*ecs.ExecuteCommandOutput, error) {
	input := &ecs.ExecuteCommandInput{
		Cluster:     clusterArn,
		Task:        taskArn,
		Container:   containerName,
		Command:     &command,
		Interactive: interactive,
	}

	output, err := c.client.ExecuteCommand(ctx, input)
	if err != nil {
		return nil, fmt.Errorf("failed to execute command: %w", err)
	}

	return output, nil
}
