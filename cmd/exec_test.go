package cmd

import (
	"context"
	"errors"
	"os"
	"os/exec"
	"testing"

	"github.com/aws/aws-sdk-go-v2/service/ecs"
	"github.com/aws/aws-sdk-go-v2/service/ecs/types"
	"github.com/sestrella/iecs/selector"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/mock"
)

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

func (m *MockSelectors) RunContainerSelector(
	ctx context.Context,
) (*selector.SelectedContainer, error) {
	args := m.Called(ctx)
	if args.Get(0) == nil {
		return nil, args.Error(1)
	}
	return args.Get(0).(*selector.SelectedContainer), args.Error(1)
}

func (m *MockSelectors) RunContainerDefinitionSelector(
	ctx context.Context,
) (*selector.SelectedContainerDefinition, error) {
	args := m.Called(ctx)
	if args.Get(0) == nil {
		return nil, args.Error(1)
	}
	return args.Get(0).(*selector.SelectedContainerDefinition), args.Error(1)
}

// Helper function to create string pointers
func stringPtr(s string) *string {
	return &s
}

func TestRunExec_Success(t *testing.T) {
	// Create mock objects
	mockSel := new(MockSelectors)

	// Setup mock responses
	clusterArn := "arn:aws:ecs:us-east-1:123456789012:cluster/my-cluster"
	clusterName := "my-cluster"
	serviceArn := "arn:aws:ecs:us-east-1:123456789012:service/my-cluster/my-service"
	taskArn := "arn:aws:ecs:us-east-1:123456789012:task/my-cluster/12345678-1234-1234-1234-123456789012"
	containerName := "my-container"
	containerRuntimeId := "12345678abcdef"

	// Mock cluster
	cluster := &types.Cluster{
		ClusterArn:  &clusterArn,
		ClusterName: &clusterName,
	}

	// Mock service
	service := &types.Service{
		ServiceArn: &serviceArn,
	}

	// Mock task
	task := &types.Task{
		TaskArn: &taskArn,
	}

	// Mock container
	container := &types.Container{
		Name:      &containerName,
		RuntimeId: &containerRuntimeId,
	}

	// Mock selected container
	selectedContainer := &selector.SelectedContainer{
		Cluster:   cluster,
		Service:   service,
		Task:      task,
		Container: container,
	}
	mockSel.On("RunContainerSelector", mock.Anything).Return(selectedContainer, nil)

	// Mock ExecuteCommand function
	mockEcsExecuteCommandFn := func(ctx context.Context, params *ecs.ExecuteCommandInput, optFns ...func(*ecs.Options)) (*ecs.ExecuteCommandOutput, error) {
		assert.Equal(t, clusterArn, *params.Cluster)
		assert.Equal(t, taskArn, *params.Task)
		assert.Equal(t, containerName, *params.Container)
		assert.Equal(t, "/bin/bash", *params.Command)
		assert.True(t, params.Interactive)

		return &ecs.ExecuteCommandOutput{
			Session: &types.Session{
				SessionId:  stringPtr("session-id"),
				StreamUrl:  stringPtr("wss://session.example.com"),
				TokenValue: stringPtr("token-value"),
			},
		}, nil
	}

	mockCommandExecutorFn := func(name string, args ...string) *exec.Cmd {
		assert.Equal(t, "session-manager-plugin", name)
		return exec.Command("echo", "test") // Use a real command that exists
	}

	// Test the function
	err := runExec(
		context.Background(),
		"session-manager-plugin",
		mockEcsExecuteCommandFn,
		mockCommandExecutorFn,
		mockSel,
		"us-east-1",
		"/bin/bash",
		true,
	)

	// Check assertions
	assert.NoError(t, err)
	mockSel.AssertExpectations(t)
}

func TestRunExec_ClusterSelectorError(t *testing.T) {
	// Create mock objects
	mockSel := new(MockSelectors)

	// Setup mock responses with an error
	expectedErr := errors.New("container selector error")
	mockSel.On("RunContainerSelector", mock.Anything).Return(nil, expectedErr)

	// Mock ExecuteCommand function - should not be called
	mockEcsExecuteCommandFn := func(ctx context.Context, params *ecs.ExecuteCommandInput, optFns ...func(*ecs.Options)) (*ecs.ExecuteCommandOutput, error) {
		t.Fatal("ExecuteCommand should not be called")
		return nil, nil
	}

	// Mock command executor function - should not be called
	mockCommandExecutorFn := func(name string, args ...string) *exec.Cmd {
		t.Fatal("Command should not be called")
		return nil
	}

	// Test the function
	err := runExec(
		context.Background(),
		"/usr/local/bin/session-manager-plugin",
		mockEcsExecuteCommandFn,
		mockCommandExecutorFn,
		mockSel,
		"us-east-1",
		"/bin/bash",
		true,
	)

	// Check assertions
	assert.Error(t, err)
	assert.Equal(t, expectedErr, err)
	mockSel.AssertExpectations(t)
}

func TestRunExec_ServiceSelectorError(t *testing.T) {
	// Create mock objects
	mockSel := new(MockSelectors)

	// Setup mock responses
	// Mock container selector error
	expectedErr := errors.New("service selector error")
	mockSel.On("RunContainerSelector", mock.Anything).Return(nil, expectedErr)

	// Mock ExecuteCommand function - should not be called
	mockEcsExecuteCommandFn := func(ctx context.Context, params *ecs.ExecuteCommandInput, optFns ...func(*ecs.Options)) (*ecs.ExecuteCommandOutput, error) {
		t.Fatal("ExecuteCommand should not be called")
		return nil, nil
	}

	// Mock command executor function - should not be called
	mockCommandExecutorFn := func(name string, args ...string) *exec.Cmd {
		t.Fatal("Command should not be called")
		return nil
	}

	// Test the function
	err := runExec(
		context.Background(),
		"/usr/local/bin/session-manager-plugin",
		mockEcsExecuteCommandFn,
		mockCommandExecutorFn,
		mockSel,
		"us-east-1",
		"/bin/bash",
		true,
	)

	// Check assertions
	assert.Error(t, err)
	assert.Equal(t, expectedErr, err)
	mockSel.AssertExpectations(t)
}

func TestRunExec_TaskSelectorError(t *testing.T) {
	// Create mock objects
	mockSel := new(MockSelectors)

	// Setup mock responses with an error
	expectedErr := errors.New("task selector error")
	mockSel.On("RunContainerSelector", mock.Anything).Return(nil, expectedErr)

	// Mock ExecuteCommand function - should not be called
	mockEcsExecuteCommandFn := func(ctx context.Context, params *ecs.ExecuteCommandInput, optFns ...func(*ecs.Options)) (*ecs.ExecuteCommandOutput, error) {
		t.Fatal("ExecuteCommand should not be called")
		return nil, nil
	}

	// Mock command executor function - should not be called
	mockCommandExecutorFn := func(name string, args ...string) *exec.Cmd {
		t.Fatal("Command should not be called")
		return nil
	}

	// Test the function
	err := runExec(
		context.Background(),
		"/usr/local/bin/session-manager-plugin",
		mockEcsExecuteCommandFn,
		mockCommandExecutorFn,
		mockSel,
		"us-east-1",
		"/bin/bash",
		true,
	)

	// Check assertions
	assert.Error(t, err)
	assert.Equal(t, expectedErr, err)
	mockSel.AssertExpectations(t)
}

func TestRunExec_ContainerSelectorError(t *testing.T) {
	// Create mock objects
	mockSel := new(MockSelectors)

	// Setup mock responses with an error
	expectedErr := errors.New("container selector error")
	mockSel.On("RunContainerSelector", mock.Anything).Return(nil, expectedErr)

	// Mock ExecuteCommand function - should not be called
	mockEcsExecuteCommandFn := func(ctx context.Context, params *ecs.ExecuteCommandInput, optFns ...func(*ecs.Options)) (*ecs.ExecuteCommandOutput, error) {
		t.Fatal("ExecuteCommand should not be called")
		return nil, nil
	}

	// Mock command executor function - should not be called
	mockCommandExecutorFn := func(name string, args ...string) *exec.Cmd {
		t.Fatal("Command should not be called")
		return nil
	}

	// Test the function
	err := runExec(
		context.Background(),
		"/usr/local/bin/session-manager-plugin",
		mockEcsExecuteCommandFn,
		mockCommandExecutorFn,
		mockSel,
		"us-east-1",
		"/bin/bash",
		true,
	)

	// Check assertions
	assert.Error(t, err)
	assert.Equal(t, expectedErr, err)
	mockSel.AssertExpectations(t)
}

func TestRunExec_ExecuteCommandError(t *testing.T) {
	// Create mock objects
	mockSel := new(MockSelectors)

	// Setup mock responses
	clusterArn := "arn:aws:ecs:us-east-1:123456789012:cluster/my-cluster"
	clusterName := "my-cluster"
	serviceArn := "arn:aws:ecs:us-east-1:123456789012:service/my-cluster/my-service"
	taskArn := "arn:aws:ecs:us-east-1:123456789012:task/my-cluster/12345678-1234-1234-1234-123456789012"
	containerName := "my-container"
	containerRuntimeId := "12345678abcdef"

	// Mock cluster
	cluster := &types.Cluster{
		ClusterArn:  &clusterArn,
		ClusterName: &clusterName,
	}

	// Mock service
	service := &types.Service{
		ServiceArn: &serviceArn,
	}

	// Mock task
	task := &types.Task{
		TaskArn: &taskArn,
	}

	// Mock container
	container := &types.Container{
		Name:      &containerName,
		RuntimeId: &containerRuntimeId,
	}

	// Mock selected container
	selectedContainer := &selector.SelectedContainer{
		Cluster:   cluster,
		Service:   service,
		Task:      task,
		Container: container,
	}
	mockSel.On("RunContainerSelector", mock.Anything).Return(selectedContainer, nil)

	// Mock ExecuteCommand error
	expectedErr := errors.New("execute command error")
	mockEcsExecuteCommandFn := func(ctx context.Context, params *ecs.ExecuteCommandInput, optFns ...func(*ecs.Options)) (*ecs.ExecuteCommandOutput, error) {
		return nil, expectedErr
	}

	// Mock command executor function - should not be called
	mockCommandExecutorFn := func(name string, args ...string) *exec.Cmd {
		t.Fatal("Command should not be called")
		return nil
	}

	// Test the function
	err := runExec(
		context.Background(),
		"/usr/local/bin/session-manager-plugin",
		mockEcsExecuteCommandFn,
		mockCommandExecutorFn,
		mockSel,
		"us-east-1",
		"/bin/bash",
		true,
	)

	// Check assertions
	assert.Error(t, err)
	assert.Equal(t, expectedErr, err)
	mockSel.AssertExpectations(t)
}

// TestHelperProcess is used by the patchExecCommand function
// to provide a stub for exec.Command
func TestHelperProcess(t *testing.T) {
	if os.Getenv("GO_WANT_HELPER_PROCESS") != "1" {
		return
	}
	os.Exit(0)
}
