package cmd

import (
	"context"
	"os/exec"
	"testing"

	"github.com/aws/aws-sdk-go-v2/service/ecs/types"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/mock"
)

func TestRunExec(t *testing.T) {
	// Create mock objects
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
		ExecSelection{
			cluster,
			service,
			task,
			container,
		},
		"/bin/bash",
		true,
	)

	// Check assertions
	assert.NoError(t, err)
	mockClient.AssertExpectations(t)
}
