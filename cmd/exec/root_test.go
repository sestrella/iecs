package exec

import (
	"context"
	"errors"
	"os"
	"os/exec"
	"testing"

	"github.com/aws/aws-sdk-go-v2/service/ecs/types"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/mock"
)

func TestRunExec_Success(t *testing.T) {
	// Create mock objects
	mockSel := new(MockSelectors)
	mockClient := new(MockClient)

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

	// Mock container
	container := &types.Container{
		Name:      &containerName,
		RuntimeId: &containerRuntimeId,
	}

	// Mock task
	task := &types.Task{
		TaskArn: &taskArn,
		Containers: []types.Container{
			*container,
		},
	}

	// Setup mock calls for selectors
	mockSel.On("Cluster", mock.Anything).Return(cluster, nil)
	mockSel.On("Service", mock.Anything, cluster).Return(service, nil)
	mockSel.On("Task", mock.Anything, service).Return(task, nil)
	mockSel.On("Container", mock.Anything, task.Containers).Return(container, nil)

	// Mock ExecuteCommand response

	mockClient.On("ExecuteCommand",
		mock.Anything,
		cluster,
		*task.TaskArn,
		container,
		"/bin/bash",
		true,
	).Return(exec.Command("echo"), nil)

	// Test the function
	err := runExec(
		context.Background(),
		mockClient,
		mockSel,
		"/bin/bash",
		true,
	)

	// Check assertions
	assert.NoError(t, err)
	mockSel.AssertExpectations(t)
	mockClient.AssertExpectations(t)
}

func TestRunExec_ClusterSelectorError(t *testing.T) {
	// Create mock objects
	mockSel := new(MockSelectors)
	mockClient := new(MockClient)

	// Setup mock responses with an error
	expectedErr := errors.New("cluster selector error")
	mockSel.On("Cluster", mock.Anything).Return(nil, expectedErr)

	// Test the function
	err := runExec(
		context.Background(),
		mockClient,
		mockSel,
		"/bin/bash",
		true,
	)

	// Check assertions
	assert.Error(t, err)
	assert.Equal(t, expectedErr, err)
	mockSel.AssertExpectations(t)
	// mockClient's ExecuteCommand should not be called
	mockClient.AssertNotCalled(
		t,
		"ExecuteCommand",
		mock.Anything,
		mock.Anything,
		mock.Anything,
		mock.Anything,
		mock.Anything,
		mock.Anything,
	)
}

func TestRunExec_ServiceSelectorError(t *testing.T) {
	// Create mock objects
	mockSel := new(MockSelectors)
	mockClient := new(MockClient)

	// Setup mock responses
	// Mock service selector error
	cluster := &types.Cluster{}
	expectedErr := errors.New("service selector error")
	mockSel.On("Cluster", mock.Anything).Return(cluster, nil)
	mockSel.On("Service", mock.Anything, cluster).Return(nil, expectedErr)

	// Test the function
	err := runExec(
		context.Background(),
		mockClient,
		mockSel,
		"/bin/bash",
		true,
	)

	// Check assertions
	assert.Error(t, err)
	assert.Equal(t, expectedErr, err)
	mockSel.AssertExpectations(t)
	// mockClient's ExecuteCommand should not be called
	mockClient.AssertNotCalled(
		t,
		"ExecuteCommand",
		mock.Anything,
		mock.Anything,
		mock.Anything,
		mock.Anything,
		mock.Anything,
		mock.Anything,
	)
}

func TestRunExec_TaskSelectorError(t *testing.T) {
	// Create mock objects
	mockSel := new(MockSelectors)
	mockClient := new(MockClient)

	// Setup mock responses with an error
	cluster := &types.Cluster{}
	service := &types.Service{}
	expectedErr := errors.New("task selector error")
	mockSel.On("Cluster", mock.Anything).Return(cluster, nil)
	mockSel.On("Service", mock.Anything, cluster).Return(service, nil)
	mockSel.On("Task", mock.Anything, service).Return(nil, expectedErr)

	// Test the function
	err := runExec(
		context.Background(),
		mockClient,
		mockSel,
		"/bin/bash",
		true,
	)

	// Check assertions
	assert.Error(t, err)
	assert.Equal(t, expectedErr, err)
	mockSel.AssertExpectations(t)
	// mockClient's ExecuteCommand should not be called
	mockClient.AssertNotCalled(
		t,
		"ExecuteCommand",
		mock.Anything,
		mock.Anything,
		mock.Anything,
		mock.Anything,
		mock.Anything,
		mock.Anything,
	)
}

func TestRunExec_ContainerSelectorError(t *testing.T) {
	// Create mock objects
	mockSel := new(MockSelectors)
	mockClient := new(MockClient)

	// Setup mock responses with an error
	cluster := &types.Cluster{}
	service := &types.Service{}
	task := &types.Task{}
	expectedErr := errors.New("container selector error")
	mockSel.On("Cluster", mock.Anything).Return(cluster, nil)
	mockSel.On("Service", mock.Anything, cluster).Return(service, nil)
	mockSel.On("Task", mock.Anything, service).Return(task, nil)
	mockSel.On("Container", mock.Anything, task.Containers).Return(nil, expectedErr)

	// Test the function
	err := runExec(
		context.Background(),
		mockClient,
		mockSel,
		"/bin/bash",
		true,
	)

	// Check assertions
	assert.Error(t, err)
	assert.Equal(t, expectedErr, err)
	mockSel.AssertExpectations(t)
	// mockClient's ExecuteCommand should not be called
	mockClient.AssertNotCalled(
		t,
		"ExecuteCommand",
		mock.Anything,
		mock.Anything,
		mock.Anything,
		mock.Anything,
		mock.Anything,
		mock.Anything,
	)
}

func TestRunExec_ExecuteCommandError(t *testing.T) {
	// Create mock objects
	mockSel := new(MockSelectors)
	mockClient := new(MockClient)

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

	// Mock container
	container := &types.Container{
		Name:      &containerName,
		RuntimeId: &containerRuntimeId,
	}

	// Mock task
	task := &types.Task{
		TaskArn: &taskArn,
		Containers: []types.Container{
			*container,
		},
	}

	// Setup mock calls for selectors
	mockSel.On("Cluster", mock.Anything).Return(cluster, nil)
	mockSel.On("Service", mock.Anything, cluster).Return(service, nil)
	mockSel.On("Task", mock.Anything, service).Return(task, nil)
	mockSel.On("Container", mock.Anything, task.Containers).Return(container, nil)

	// Mock ExecuteCommand error
	expectedErr := errors.New("execute command error")
	mockClient.On("ExecuteCommand",
		mock.Anything,
		cluster,
		*task.TaskArn,
		container,
		"/bin/bash",
		true,
	).Return(exec.Command("echo"), expectedErr)

	// Test the function
	err := runExec(
		context.Background(),
		mockClient,
		mockSel,
		"/bin/bash",
		true,
	)

	// Check assertions
	assert.Error(t, err)
	assert.Equal(t, expectedErr, err)
	mockSel.AssertExpectations(t)
	mockClient.AssertExpectations(t)
}

// TestHelperProcess is used by the patchExecCommand function
// to provide a stub for exec.Command
func TestHelperProcess(t *testing.T) {
	if os.Getenv("GO_WANT_HELPER_PROCESS") != "1" {
		return
	}
	os.Exit(0)
}
