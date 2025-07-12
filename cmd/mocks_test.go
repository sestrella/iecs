package cmd

import (
	"context"
	"os/exec"

	"github.com/aws/aws-sdk-go-v2/service/ecs/types"
	"github.com/sestrella/iecs/client"
	"github.com/stretchr/testify/mock"
)

// MockClient mocks the client.Client interface
type MockClient struct {
	mock.Mock
}

func (m *MockClient) ListClusters(ctx context.Context) ([]string, error) {
	args := m.Called(ctx)
	if args.Get(0) == nil {
		return nil, args.Error(1)
	}
	return args.Get(0).([]string), args.Error(1)
}

func (m *MockClient) DescribeClusters(
	ctx context.Context,
	clusterArns []string,
) ([]types.Cluster, error) {
	args := m.Called(ctx, clusterArns)
	if args.Get(0) == nil {
		return nil, args.Error(1)
	}
	return args.Get(0).([]types.Cluster), args.Error(1)
}

func (m *MockClient) ListServices(ctx context.Context, clusterArn string) ([]string, error) {
	args := m.Called(ctx, clusterArn)
	if args.Get(0) == nil {
		return nil, args.Error(1)
	}
	return args.Get(0).([]string), args.Error(1)
}

func (m *MockClient) DescribeServices(
	ctx context.Context,
	clusterArn string,
	serviceArns []string,
) ([]types.Service, error) {
	args := m.Called(ctx, clusterArn, serviceArns)
	if args.Get(0) == nil {
		return nil, args.Error(1)
	}
	return args.Get(0).([]types.Service), args.Error(1)
}

func (m *MockClient) ListTasks(
	ctx context.Context,
	clusterArn string,
	serviceArn string,
) ([]string, error) {
	args := m.Called(ctx, clusterArn, serviceArn)
	if args.Get(0) == nil {
		return nil, args.Error(1)
	}
	return args.Get(0).([]string), args.Error(1)
}

func (m *MockClient) DescribeTasks(
	ctx context.Context,
	clusterArn string,
	taskArns []string,
) ([]types.Task, error) {
	args := m.Called(ctx, clusterArn, taskArns)
	if args.Get(0) == nil {
		return nil, args.Error(1)
	}
	return args.Get(0).([]types.Task), args.Error(1)
}

func (m *MockClient) StartLiveTail(
	ctx context.Context,
	logGroupName string,
	streamPrefix string,
	handler client.LiveTailHandlers,
) error {
	args := m.Called(ctx, logGroupName, streamPrefix, handler)
	return args.Error(0)
}

func (m *MockClient) ExecuteCommand(
	ctx context.Context,
	cluster *types.Cluster,
	taskArn string,
	container *types.Container,
	command string,
	interactive bool,
) (*exec.Cmd, error) {
	args := m.Called(ctx, cluster, taskArn, container, command, interactive)
	if args.Get(0) == nil {
		return nil, args.Error(1)
	}
	return args.Get(0).(*exec.Cmd), args.Error(1)
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

// MockSelectors mocks the selector.Selectors interface
type MockSelectors struct {
	mock.Mock
}

func (m *MockSelectors) Cluster(ctx context.Context) (*types.Cluster, error) {
	args := m.Called(ctx)
	if args.Get(0) == nil {
		return nil, args.Error(1)
	}
	return args.Get(0).(*types.Cluster), args.Error(1)
}

func (m *MockSelectors) Service(
	ctx context.Context,
	cluster *types.Cluster,
) (*types.Service, error) {
	args := m.Called(ctx, cluster)
	if args.Get(0) == nil {
		return nil, args.Error(1)
	}
	return args.Get(0).(*types.Service), args.Error(1)
}

func (m *MockSelectors) Task(
	ctx context.Context,
	service *types.Service,
) (*types.Task, error) {
	args := m.Called(ctx, service)
	if args.Get(0) == nil {
		return nil, args.Error(1)
	}
	return args.Get(0).(*types.Task), args.Error(1)
}

func (m *MockSelectors) Tasks(
	ctx context.Context,
	service *types.Service,
) ([]types.Task, error) {
	args := m.Called(ctx, service)
	if args.Get(0) == nil {
		return nil, args.Error(1)
	}
	return args.Get(0).([]types.Task), args.Error(1)
}

func (m *MockSelectors) Container(
	ctx context.Context,
	containers []types.Container,
) (*types.Container, error) {
	args := m.Called(ctx, containers)
	if args.Get(0) == nil {
		return nil, args.Error(1)
	}
	return args.Get(0).(*types.Container), args.Error(1)
}

func (m *MockSelectors) ContainerDefinitions(
	ctx context.Context,
	taskDefinitionArn string,
) ([]types.ContainerDefinition, error) {
	args := m.Called(ctx, taskDefinitionArn)
	if args.Get(0) == nil {
		return nil, args.Error(1)
	}
	return args.Get(0).([]types.ContainerDefinition), args.Error(1)
}
