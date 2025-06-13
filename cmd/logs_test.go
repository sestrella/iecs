package cmd

import (
	"context"
	"errors"
	"testing"
	"time"

	"github.com/aws/aws-sdk-go-v2/service/ecs/types"
	"github.com/sestrella/iecs/client"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/mock"
)

// The MockClient implementation is now in mocks_test.go

func TestRunLogs_Success(t *testing.T) {
	// Create mock objects
	mockClient := new(MockClient)
	mockSel := new(MockSelectors)

	// Setup mock responses
	clusterArn := "arn:aws:ecs:us-east-1:123456789012:cluster/my-cluster"
	clusterName := "my-cluster"
	serviceArn := "arn:aws:ecs:us-east-1:123456789012:service/my-cluster/my-service"
	containerDefinitionName := "my-container"

	// Mock cluster
	cluster := &types.Cluster{
		ClusterArn:  &clusterArn,
		ClusterName: &clusterName,
	}

	// Mock service
	service := &types.Service{
		ServiceArn: &serviceArn,
	}

	// Mock container definition with log configuration
	logConfiguration := &types.LogConfiguration{
		LogDriver: "awslogs",
		Options: map[string]string{
			"awslogs-group":         "/ecs/my-service",
			"awslogs-stream-prefix": "ecs",
		},
	}

	containerDefinition := &types.ContainerDefinition{
		Name:             &containerDefinitionName,
		LogConfiguration: logConfiguration,
	}

	// Setup mock calls for selectors
	mockSel.On("Cluster", mock.Anything).Return(cluster, nil)
	mockSel.On("Service", mock.Anything, cluster).Return(service, nil)
	mockSel.On("ContainerDefinition", mock.Anything, service).Return(containerDefinition, nil)
	mockClient.On("StartLiveTail", mock.Anything, "/ecs/my-service", "ecs", mock.AnythingOfType("client.EventHandler")).Return(nil)

	// Test the function
	err := runLogs(context.Background(), mockClient, mockSel)

	// Check assertions
	assert.NoError(t, err)
	mockSel.AssertExpectations(t)
	mockClient.AssertExpectations(t)
}

func TestRunLogs_SelectorError(t *testing.T) {
	// Create mock objects
	mockClient := new(MockClient)
	mockSel := new(MockSelectors)

	// Setup mock responses with an error for cluster selector
	expectedErr := errors.New("cluster selector error")
	mockSel.On("Cluster", mock.Anything).Return(nil, expectedErr)

	// Test the function
	err := runLogs(context.Background(), mockClient, mockSel)

	// Check assertions
	assert.Error(t, err)
	assert.Equal(t, expectedErr, err)
	mockSel.AssertExpectations(t)
	// StartLiveTail should not be called
	mockClient.AssertNotCalled(t, "StartLiveTail")
}

func TestRunLogs_MissingLogConfiguration(t *testing.T) {
	// Create mock objects
	mockClient := new(MockClient)
	mockSel := new(MockSelectors)

	// Setup mock responses
	clusterArn := "arn:aws:ecs:us-east-1:123456789012:cluster/my-cluster"
	clusterName := "my-cluster"
	serviceArn := "arn:aws:ecs:us-east-1:123456789012:service/my-cluster/my-service"
	containerDefinitionName := "my-container"

	// Mock cluster
	cluster := &types.Cluster{
		ClusterArn:  &clusterArn,
		ClusterName: &clusterName,
	}

	// Mock service
	service := &types.Service{
		ServiceArn: &serviceArn,
	}

	// Mock container definition with nil log configuration
	containerDefinition := &types.ContainerDefinition{
		Name:             &containerDefinitionName,
		LogConfiguration: nil,
	}

	// Setup mock calls for selectors
	mockSel.On("Cluster", mock.Anything).Return(cluster, nil)
	mockSel.On("Service", mock.Anything, cluster).Return(service, nil)
	mockSel.On("ContainerDefinition", mock.Anything, service).Return(containerDefinition, nil)

	// Test the function
	err := runLogs(context.Background(), mockClient, mockSel)

	// Check assertions
	assert.Error(t, err)
	assert.Contains(t, err.Error(), "no log configuration found")
	mockSel.AssertExpectations(t)
	// StartLiveTail should not be called
	mockClient.AssertNotCalled(t, "StartLiveTail")
}

func TestRunLogs_MissingLogOptions(t *testing.T) {
	// Create mock objects
	mockClient := new(MockClient)
	mockSel := new(MockSelectors)

	// Setup mock responses
	clusterArn := "arn:aws:ecs:us-east-1:123456789012:cluster/my-cluster"
	clusterName := "my-cluster"
	serviceArn := "arn:aws:ecs:us-east-1:123456789012:service/my-cluster/my-service"
	containerDefinitionName := "my-container"

	// Mock cluster
	cluster := &types.Cluster{
		ClusterArn:  &clusterArn,
		ClusterName: &clusterName,
	}

	// Mock service
	service := &types.Service{
		ServiceArn: &serviceArn,
	}

	// Mock container definition with log configuration but missing options
	logConfiguration := &types.LogConfiguration{
		LogDriver: "awslogs",
		Options:   map[string]string{}, // Empty options
	}

	containerDefinition := &types.ContainerDefinition{
		Name:             &containerDefinitionName,
		LogConfiguration: logConfiguration,
	}

	// Setup mock calls for selectors
	mockSel.On("Cluster", mock.Anything).Return(cluster, nil)
	mockSel.On("Service", mock.Anything, cluster).Return(service, nil)
	mockSel.On("ContainerDefinition", mock.Anything, service).Return(containerDefinition, nil)

	// Test the function
	err := runLogs(context.Background(), mockClient, mockSel)

	// Check assertions
	assert.Error(t, err)
	assert.Contains(t, err.Error(), "missing log options")
	mockSel.AssertExpectations(t)
	// StartLiveTail should not be called
	mockClient.AssertNotCalled(t, "StartLiveTail")
}

func TestRunLogs_StartLiveTailError(t *testing.T) {
	// Create mock objects
	mockClient := new(MockClient)
	mockSel := new(MockSelectors)

	// Setup mock responses
	clusterArn := "arn:aws:ecs:us-east-1:123456789012:cluster/my-cluster"
	clusterName := "my-cluster"
	serviceArn := "arn:aws:ecs:us-east-1:123456789012:service/my-cluster/my-service"
	containerDefinitionName := "my-container"

	// Mock cluster
	cluster := &types.Cluster{
		ClusterArn:  &clusterArn,
		ClusterName: &clusterName,
	}

	// Mock service
	service := &types.Service{
		ServiceArn: &serviceArn,
	}

	// Mock container definition with log configuration
	logConfiguration := &types.LogConfiguration{
		LogDriver: "awslogs",
		Options: map[string]string{
			"awslogs-group":         "/ecs/my-service",
			"awslogs-stream-prefix": "ecs",
		},
	}

	containerDefinition := &types.ContainerDefinition{
		Name:             &containerDefinitionName,
		LogConfiguration: logConfiguration,
	}

	// Setup mock calls for selectors
	mockSel.On("Cluster", mock.Anything).Return(cluster, nil)
	mockSel.On("Service", mock.Anything, cluster).Return(service, nil)
	mockSel.On("ContainerDefinition", mock.Anything, service).Return(containerDefinition, nil)

	// Setup StartLiveTail to return an error
	expectedErr := errors.New("failed to start live tail")
	mockClient.On("StartLiveTail", mock.Anything, "/ecs/my-service", "ecs", mock.AnythingOfType("client.EventHandler")).Return(expectedErr)

	// Test the function
	err := runLogs(context.Background(), mockClient, mockSel)

	// Check assertions
	assert.Error(t, err)
	assert.Equal(t, expectedErr, err)
	mockSel.AssertExpectations(t)
	mockClient.AssertExpectations(t)
}

// Test handler function behavior
func TestRunLogs_HandlerBehavior(t *testing.T) {
	// Create mock objects
	mockClient := new(MockClient)
	mockSel := new(MockSelectors)

	// Setup mock responses
	clusterArn := "arn:aws:ecs:us-east-1:123456789012:cluster/my-cluster"
	clusterName := "my-cluster"
	serviceArn := "arn:aws:ecs:us-east-1:123456789012:service/my-cluster/my-service"
	containerDefinitionName := "my-container"

	// Mock cluster
	cluster := &types.Cluster{
		ClusterArn:  &clusterArn,
		ClusterName: &clusterName,
	}

	// Mock service
	service := &types.Service{
		ServiceArn: &serviceArn,
	}

	// Mock container definition with log configuration
	logConfiguration := &types.LogConfiguration{
		LogDriver: "awslogs",
		Options: map[string]string{
			"awslogs-group":         "/ecs/my-service",
			"awslogs-stream-prefix": "ecs",
		},
	}

	containerDefinition := &types.ContainerDefinition{
		Name:             &containerDefinitionName,
		LogConfiguration: logConfiguration,
	}

	// Setup mock calls for selectors
	mockSel.On("Cluster", mock.Anything).Return(cluster, nil)
	mockSel.On("Service", mock.Anything, cluster).Return(service, nil)
	mockSel.On("ContainerDefinition", mock.Anything, service).Return(containerDefinition, nil)

	// Capture the handler function
	var capturedHandler client.EventHandler
	mockClient.On("StartLiveTail", mock.Anything, "/ecs/my-service", "ecs", mock.AnythingOfType("client.EventHandler")).Run(func(args mock.Arguments) {
		capturedHandler = args.Get(3).(client.EventHandler)
	}).Return(nil)

	// Start the logs function
	err := runLogs(context.Background(), mockClient, mockSel)
	assert.NoError(t, err)

	// Test that the function was called
	mockClient.AssertExpectations(t)

	// Ensure we captured the handler
	assert.NotNil(t, capturedHandler)

	// Create a temporary hook to capture fmt.Printf output
	// Note: In a real test, you might use a testing library like testify/assert
	// with output capture, or redirect os.Stdout
	timestamp := time.Now()
	message := "test log message"

	// Call the handler with test data
	// In a real implementation, you'd capture the output and verify it
	capturedHandler(timestamp, message)

	// Since we can't easily capture fmt.Printf output in this example,
	// we mainly verify that the handler doesn't panic and completes
	// In a full implementation, you would verify the output format
}
