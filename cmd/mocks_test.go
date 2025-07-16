package cmd

import (
	"context"

	"github.com/aws/aws-sdk-go-v2/service/ecs/types"
	"github.com/stretchr/testify/mock"
)

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

func (m *MockSelectors) TaskDefinition(
	ctx context.Context,
	serviceArn string,
) (*types.TaskDefinition, error) {
	args := m.Called(ctx, serviceArn)
	if args.Get(0) == nil {
		return nil, args.Error(1)
	}
	return args.Get(0).(*types.TaskDefinition), args.Error(1)
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
