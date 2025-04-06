package client

import (
	"context"

	"github.com/aws/aws-sdk-go-v2/service/ecs"
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
