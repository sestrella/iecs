package client

import (
	"context"

	"github.com/aws/aws-sdk-go-v2/service/ecs"
	"github.com/aws/aws-sdk-go-v2/service/ecs/types"
	"github.com/stretchr/testify/mock"
)

// MockClient implements a mock version of the Client interface for testing
type MockClient struct {
	mock.Mock
}

func (m *MockClient) DescribeCluster(
	ctx context.Context,
	clusterArn string,
) (*types.Cluster, error) {
	args := m.Called(ctx, clusterArn)
	if args.Get(0) == nil {
		return nil, args.Error(1)
	}
	return args.Get(0).(*types.Cluster), args.Error(1)
}

func (m *MockClient) DescribeService(
	ctx context.Context,
	clusterArn string,
	serviceArn string,
) (*types.Service, error) {
	args := m.Called(ctx, clusterArn, serviceArn)
	if args.Get(0) == nil {
		return nil, args.Error(1)
	}
	return args.Get(0).(*types.Service), args.Error(1)
}

func (m *MockClient) DescribeTask(
	ctx context.Context,
	clusterArn string,
	taskArn string,
) (*types.Task, error) {
	args := m.Called(ctx, clusterArn, taskArn)
	if args.Get(0) == nil {
		return nil, args.Error(1)
	}
	return args.Get(0).(*types.Task), args.Error(1)
}

func (m *MockClient) DescribeTaskDefinition(
	ctx context.Context,
	taskDefinitionArn string,
) (*types.TaskDefinition, error) {
	args := m.Called(ctx, taskDefinitionArn)
	if args.Get(0) == nil {
		return nil, args.Error(1)
	}
	return args.Get(0).(*types.TaskDefinition), args.Error(1)
}

func (m *MockClient) ListClusters(ctx context.Context) ([]string, error) {
	args := m.Called(ctx)
	return args.Get(0).([]string), args.Error(1)
}

func (m *MockClient) ListServices(ctx context.Context, clusterArn string) ([]string, error) {
	args := m.Called(ctx, clusterArn)
	return args.Get(0).([]string), args.Error(1)
}

func (m *MockClient) ListTasks(
	ctx context.Context,
	clusterArn string,
	serviceArn string,
) ([]string, error) {
	args := m.Called(ctx, clusterArn, serviceArn)
	return args.Get(0).([]string), args.Error(1)
}

func (m *MockClient) ExecuteCommand(
	ctx context.Context,
	clusterArn *string,
	taskArn *string,
	containerName *string,
	command string,
	interactive bool,
) (*ecs.ExecuteCommandOutput, error) {
	args := m.Called(ctx, clusterArn, taskArn, containerName, command, interactive)
	if args.Get(0) == nil {
		return nil, args.Error(1)
	}
	return args.Get(0).(*ecs.ExecuteCommandOutput), args.Error(1)
}

func (m *MockClient) StartLiveTail(
	ctx context.Context,
	logGroupName string,
	streamPrefix string,
	handler EventHandler,
) error {
	args := m.Called(ctx, logGroupName, streamPrefix, handler)
	return args.Error(0)
}
