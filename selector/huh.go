package selector

import (
	"context"
	"fmt"

	"github.com/aws/aws-sdk-go-v2/service/ecs"
	"github.com/aws/aws-sdk-go-v2/service/ecs/types"
	"github.com/charmbracelet/huh"
)

// SelectionResult holds the selected cluster, service, task and container
type SelectionResult struct {
	Cluster   *types.Cluster
	Service   *types.Service
	Task      *types.Task
	Container *types.Container
}

// RunHuhForm runs an interactive form to select an ECS cluster, service, task and container
func RunHuhForm(ctx context.Context, client Client) (*SelectionResult, error) {
	result := &SelectionResult{}

	// Variables to store form selections
	var selectedClusterArn string
	var selectedServiceArn string
	var selectedTaskArn string
	var selectedContainerName string

	// Get clusters
	listClusters, err := client.ListClusters(ctx, &ecs.ListClustersInput{})
	if err != nil {
		return nil, fmt.Errorf("failed to list clusters: %w", err)
	}
	if len(listClusters.ClusterArns) == 0 {
		return nil, fmt.Errorf("no clusters found")
	}

	// Create the form
	form := huh.NewForm(
		huh.NewGroup(
			huh.NewSelect[string]().
				Title("Select Cluster").
				Options(
					huh.NewOptions(listClusters.ClusterArns...)...,
				).
				Value(&selectedClusterArn),
		),
	)

	// Run the form to select cluster
	err = form.Run()
	if err != nil {
		return nil, fmt.Errorf("form error: %w", err)
	}

	// Get selected cluster details
	describeClusters, err := client.DescribeClusters(ctx, &ecs.DescribeClustersInput{
		Clusters: []string{selectedClusterArn},
	})
	if err != nil {
		return nil, fmt.Errorf("failed to describe cluster: %w", err)
	}
	if len(describeClusters.Clusters) == 0 {
		return nil, fmt.Errorf("cluster not found: %s", selectedClusterArn)
	}
	result.Cluster = &describeClusters.Clusters[0]

	// Get services for selected cluster
	listServices, err := client.ListServices(ctx, &ecs.ListServicesInput{
		Cluster: &selectedClusterArn,
	})
	if err != nil {
		return nil, fmt.Errorf("failed to list services: %w", err)
	}
	if len(listServices.ServiceArns) == 0 {
		return nil, fmt.Errorf("no services found in cluster: %s", selectedClusterArn)
	}

	// Create service selection form
	serviceForm := huh.NewForm(
		huh.NewGroup(
			huh.NewSelect[string]().
				Title("Select Service").
				Options(
					huh.NewOptions(listServices.ServiceArns...)...,
				).
				Value(&selectedServiceArn),
		),
	)

	// Run service selection
	err = serviceForm.Run()
	if err != nil {
		return nil, fmt.Errorf("form error: %w", err)
	}

	// Get selected service details
	describeService, err := client.DescribeServices(ctx, &ecs.DescribeServicesInput{
		Cluster:  &selectedClusterArn,
		Services: []string{selectedServiceArn},
	})
	if err != nil {
		return nil, fmt.Errorf("failed to describe service: %w", err)
	}
	if len(describeService.Services) == 0 {
		return nil, fmt.Errorf("service not found: %s", selectedServiceArn)
	}
	result.Service = &describeService.Services[0]

	// Get tasks for selected service
	listTasks, err := client.ListTasks(ctx, &ecs.ListTasksInput{
		Cluster:     &selectedClusterArn,
		ServiceName: result.Service.ServiceName,
	})
	if err != nil {
		return nil, fmt.Errorf("failed to list tasks: %w", err)
	}
	if len(listTasks.TaskArns) == 0 {
		return nil, fmt.Errorf("no tasks found for service: %s", *result.Service.ServiceName)
	}

	// Create task selection form
	taskForm := huh.NewForm(
		huh.NewGroup(
			huh.NewSelect[string]().
				Title("Select Task").
				Options(
					huh.NewOptions(listTasks.TaskArns...)...,
				).
				Value(&selectedTaskArn),
		),
	)

	// Run task selection
	err = taskForm.Run()
	if err != nil {
		return nil, fmt.Errorf("form error: %w", err)
	}

	// Get selected task details
	describeTasks, err := client.DescribeTasks(ctx, &ecs.DescribeTasksInput{
		Cluster: &selectedClusterArn,
		Tasks:   []string{selectedTaskArn},
	})
	if err != nil {
		return nil, fmt.Errorf("failed to describe task: %w", err)
	}
	if len(describeTasks.Tasks) == 0 {
		return nil, fmt.Errorf("task not found: %s", selectedTaskArn)
	}
	result.Task = &describeTasks.Tasks[0]

	// Build container options
	var containerNames []string
	for _, container := range result.Task.Containers {
		containerNames = append(containerNames, *container.Name)
	}

	if len(containerNames) == 0 {
		return nil, fmt.Errorf("no containers found for task: %s", selectedTaskArn)
	}

	// Create container selection form
	containerForm := huh.NewForm(
		huh.NewGroup(
			huh.NewSelect[string]().
				Title("Select Container").
				Options(
					huh.NewOptions(containerNames...)...,
				).
				Value(&selectedContainerName),
		),
	)

	// Run container selection
	err = containerForm.Run()
	if err != nil {
		return nil, fmt.Errorf("form error: %w", err)
	}

	// Find selected container
	for _, container := range result.Task.Containers {
		if *container.Name == selectedContainerName {
			result.Container = &container
			break
		}
	}

	if result.Container == nil {
		return nil, fmt.Errorf("container not found: %s", selectedContainerName)
	}

	return result, nil
}
