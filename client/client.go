package client

import (
	"context"
	"os/exec"
	"time"

	logsTypes "github.com/aws/aws-sdk-go-v2/service/cloudwatchlogs/types"
	ecsTypes "github.com/aws/aws-sdk-go-v2/service/ecs/types"
)

// EventHandler is a function that handles log events.
type LiveTailHandlers struct {
	Start  func()
	Update func(logsTypes.LiveTailSessionLogEvent)
}

type UpdateServiceInput struct {
	TaskDefinitionArn string
	DesiredCounts     int
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
	UpdateService(
		ctx context.Context,
		service ecsTypes.Service,
		input UpdateServiceInput,
		waitTimeout time.Duration,
	) (*ecsTypes.Service, error)

	// Tasks
	ListTasks(ctx context.Context, clusterArn string, serviceArn string) ([]string, error)
	DescribeTasks(
		ctx context.Context,
		clusterArn string,
		taskArns []string,
	) ([]ecsTypes.Task, error)

	// Task Definitions
	ListTaskDefinitions(
		ctx context.Context,
		familyPrefix string,
	) ([]string, error)
	DescribeTaskDefinition(
		ctx context.Context,
		taskDefinitionArn string,
	) (*ecsTypes.TaskDefinition, error)

	// Others
	ExecuteCommand(
		ctx context.Context,
		cluster *ecsTypes.Cluster,
		taskArn string,
		container *ecsTypes.Container,
		command string,
		interactive bool,
	) (*exec.Cmd, error)
	StartLiveTail(
		ctx context.Context,
		logGroupName string,
		streamPrefix string,
		handler LiveTailHandlers,
	) error
}
