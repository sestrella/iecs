package selector

import (
	"context"
	"regexp"
	"testing"

	"github.com/aws/aws-sdk-go-v2/service/ecs/types"
	"github.com/charmbracelet/huh"
	"github.com/sestrella/iecs/test"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/mock"
)

func TestCluster(t *testing.T) {
	t.Run("clusterRegex is nil", func(t *testing.T) {
		clusterArn := "arn:aws:ecs:us-east-1:123456789012:cluster/my-cluster"
		cluster := types.Cluster{
			ClusterArn: &clusterArn,
		}

		mockClient := new(test.MockClient)

		mockClient.On("ListClusters", mock.Anything).Return([]string{clusterArn}, nil)
		mockClient.On("DescribeClusters", mock.Anything, []string{clusterArn}).
			Return([]types.Cluster{cluster}, nil)

		selectors := NewSelectors(mockClient, *huh.ThemeBase())
		selectedCluster, err := selectors.Cluster(context.Background(), nil)

		assert.NoError(t, err)
		assert.Equal(t, &cluster, selectedCluster)
	})

	t.Run("clusterRegex is not nil", func(t *testing.T) {
		clusterArn1 := "arn:aws:ecs:us-east-1:123456789012:cluster/my-cluster-1"
		clusterArn2 := "arn:aws:ecs:us-east-1:123456789012:cluster/my-cluster-2"

		cluster1 := types.Cluster{
			ClusterArn: &clusterArn1,
		}

		mockClient := new(test.MockClient)

		mockClient.On("ListClusters", mock.Anything).Return([]string{clusterArn1, clusterArn2}, nil)
		mockClient.On("DescribeClusters", mock.Anything, []string{clusterArn1}).
			Return([]types.Cluster{cluster1}, nil)

		selectors := NewSelectors(mockClient, *huh.ThemeBase())
		selectedCluster, err := selectors.Cluster(
			context.Background(),
			regexp.MustCompile("my-cluster-1"),
		)

		assert.NoError(t, err)
		assert.Equal(t, &cluster1, selectedCluster)
	})
}

func TestService(t *testing.T) {
	clusterArn := "arn:aws:ecs:us-east-1:123456789012:cluster/my-cluster"

	cluster := types.Cluster{
		ClusterArn: &clusterArn,
	}

	t.Run("serviceRegex is nil", func(t *testing.T) {
		serviceArn := "arn:aws:ecs:us-east-1:123456789012:service/my-cluster/my-service"

		service := types.Service{
			ServiceArn: &serviceArn,
		}

		mockClient := new(test.MockClient)

		mockClient.On("ListServices", mock.Anything, clusterArn).Return([]string{serviceArn}, nil)
		mockClient.On("DescribeServices", mock.Anything, clusterArn, []string{serviceArn}).
			Return([]types.Service{service}, nil)

		selectors := NewSelectors(mockClient, *huh.ThemeBase())
		selectedService, err := selectors.Service(context.Background(), &cluster, nil)

		assert.NoError(t, err)
		assert.Equal(t, &service, selectedService)
	})

	t.Run("serviceRegex is not nil", func(t *testing.T) {
		serviceArn1 := "arn:aws:ecs:us-east-1:123456789012:service/my-cluster/my-service-1"
		serviceArn2 := "arn:aws:ecs:us-east-1:123456789012:service/my-cluster/my-service-2"

		service1 := types.Service{
			ServiceArn: &serviceArn1,
		}

		mockClient := new(test.MockClient)

		mockClient.On("ListServices", mock.Anything, clusterArn).
			Return([]string{serviceArn1, serviceArn2}, nil)
		mockClient.On("DescribeServices", mock.Anything, clusterArn, []string{serviceArn1}).
			Return([]types.Service{service1}, nil)

		selectors := NewSelectors(mockClient, *huh.ThemeBase())
		selectedService, err := selectors.Service(
			context.Background(),
			&cluster,
			regexp.MustCompile("my-service-1"),
		)

		assert.NoError(t, err)
		assert.Equal(t, &service1, selectedService)
	})
}

func TestTask(t *testing.T) {
	clusterArn := "arn:aws:ecs:us-east-1:123456789012:cluster/my-cluster"

	serviceArn := "arn:aws:ecs:us-east-1:123456789012:service/my-cluster/my-service"

	service := types.Service{
		ClusterArn: &clusterArn,
		ServiceArn: &serviceArn,
	}

	t.Run("taskRegex is nil", func(t *testing.T) {
		taskArn := "arn:aws:ecs:us-east-1:123456789012:task/my-cluster/12345678-1234-1234-1234-123456789012"

		task := types.Task{
			TaskArn: &taskArn,
		}

		mockClient := new(test.MockClient)

		mockClient.On("ListTasks", mock.Anything, clusterArn, serviceArn).
			Return([]string{taskArn}, nil)
		mockClient.On("DescribeTasks", mock.Anything, clusterArn, []string{taskArn}).
			Return([]types.Task{task}, nil)

		selectors := NewSelectors(mockClient, *huh.ThemeBase())
		selectedTask, err := selectors.Task(context.Background(), &service, nil)

		assert.NoError(t, err)
		assert.Equal(t, &task, selectedTask)
	})

	t.Run("taskRegex is not nil", func(t *testing.T) {
		taskArn1 := "arn:aws:ecs:us-east-1:123456789012:task/my-cluster/12345678-1234-1234-1234-123456789012"
		taskArn2 := "arn:aws:ecs:us-east-1:123456789012:task/my-cluster/12345678-1234-1234-1234-123456789013"

		task1 := types.Task{
			TaskArn: &taskArn1,
		}

		mockClient := new(test.MockClient)

		mockClient.On("ListTasks", mock.Anything, clusterArn, serviceArn).
			Return([]string{taskArn1, taskArn2}, nil)
		mockClient.On("DescribeTasks", mock.Anything, clusterArn, []string{taskArn1}).
			Return([]types.Task{task1}, nil)

		selectors := NewSelectors(mockClient, *huh.ThemeBase())
		selectedTask, err := selectors.Task(
			context.Background(),
			&service,
			regexp.MustCompile("1234-123456789012"),
		)

		assert.NoError(t, err)
		assert.Equal(t, &task1, selectedTask)
	})
}
