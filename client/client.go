package client

import (
	"context"
	"os/exec"

	logsTypes "github.com/aws/aws-sdk-go-v2/service/cloudwatchlogs/types"
	ecsTypes "github.com/aws/aws-sdk-go-v2/service/ecs/types"
)

// EventHandler is a function that handles log events.
type LiveTailHandlers struct {
	Start  func()
	Update func(logsTypes.LiveTailSessionLogEvent)
}

// Client interface combines ECS and CloudWatch Logs operations.
type Client interface {
	// Clusters
	ListClusters(ctx context.Context) ([]string, error)
	DescribeClusters(ctx context.Context, clusterArns []string) ([]ecsTypes.Cluster, error)

	// Services
	ListServices(ctx context.Context, clusterArn string) ([]string, error)
	DescribeServices(
		ctx context.Context,
		clusterArn string,
		serviceArns []string,
	) ([]ecsTypes.Service, error)

	// Tasks
	ListTasks(ctx context.Context, clusterArn string, serviceArn string) ([]string, error)
	DescribeTasks(
		ctx context.Context,
		clusterArn string,
		taskArns []string,
	) ([]ecsTypes.Task, error)

	// CloudWatch Logs operations
	StartLiveTail(
		ctx context.Context,
		logGroupName string,
		streamPrefix string,
		handler LiveTailHandlers,
	) error

	// ECS operations
	ExecuteCommand(
		ctx context.Context,
		cluster *ecsTypes.Cluster,
		taskArn string,
		container *ecsTypes.Container,
		command string,
		interactive bool,
	) (*exec.Cmd, error)

	// Task Definitions
	DescribeTaskDefinition(
		ctx context.Context,
		taskDefinitionArn string,
	) (*ecsTypes.TaskDefinition, error)
}
