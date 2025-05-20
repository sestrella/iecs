package cmd

import (
	"context"
	"github.com/aws/aws-sdk-go-v2/service/ecs"
	"github.com/aws/aws-sdk-go-v2/service/ecs/types"
	"github.com/sestrella/iecs/client"
	"github.com/sestrella/iecs/selector"
	"github.com/stretchr/testify/mock"
)

// MockClient mocks the client.Client interface
type MockClient struct {
	mock.Mock
}

func (m *MockClient) StartLiveTail(
	ctx context.Context,
	logGroupName string,
	streamPrefix string,
	handler client.EventHandler,
) error {
	args := m.Called(ctx, logGroupName, streamPrefix, handler)
	return args.Error(0)
}

func (m *MockClient) ExecuteCommand(
	ctx context.Context,
	cluster string,
	task string,
	container string,
	command string,
	interactive bool,
) (*ecs.ExecuteCommandOutput, error) {
	args := m.Called(ctx, cluster, task, container, command, interactive)
	if args.Get(0) == nil {
		return nil, args.Error(1)
	}
	return args.Get(0).(*ecs.ExecuteCommandOutput), args.Error(1)
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

func (m *MockSelectors) Service(ctx context.Context, clusterArn string) (*types.Service, error) {
	args := m.Called(ctx, clusterArn)
	if args.Get(0) == nil {
		return nil, args.Error(1)
	}
	return args.Get(0).(*types.Service), args.Error(1)
}

func (m *MockSelectors) Task(
	ctx context.Context,
	clusterArn string,
	serviceArn string,
) (*types.Task, error) {
	args := m.Called(ctx, clusterArn, serviceArn)
	if args.Get(0) == nil {
		return nil, args.Error(1)
	}
	return args.Get(0).(*types.Task), args.Error(1)
}

func (m *MockSelectors) Container(containers []types.Container) (*types.Container, error) {
	args := m.Called(containers)
	if args.Get(0) == nil {
		return nil, args.Error(1)
	}
	return args.Get(0).(*types.Container), args.Error(1)
}

func (m *MockSelectors) ContainerDefinition(
	ctx context.Context,
	taskDefinition string,
) (*types.ContainerDefinition, error) {
	args := m.Called(ctx, taskDefinition)
	if args.Get(0) == nil {
		return nil, args.Error(1)
	}
	return args.Get(0).(*types.ContainerDefinition), args.Error(1)
}

func (m *MockSelectors) ContainerSelector(
	ctx context.Context,
) (*selector.SelectedContainer, error) {
	args := m.Called(ctx)
	if args.Get(0) == nil {
		return nil, args.Error(1)
	}
	return args.Get(0).(*selector.SelectedContainer), args.Error(1)
}

func (m *MockSelectors) ContainerDefinitionSelector(
	ctx context.Context,
) (*selector.SelectedContainerDefinition, error) {
	args := m.Called(ctx)
	if args.Get(0) == nil {
		return nil, args.Error(1)
	}
	return args.Get(0).(*selector.SelectedContainerDefinition), args.Error(1)
}
