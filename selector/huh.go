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

	// Create the form with all selections
	form := huh.NewForm(
		huh.NewGroup(
			huh.NewSelect[string]().
				Title("Select Cluster").
				Options(
					huh.NewOptions(listClusters.ClusterArns...)...,
				).
				Value(&selectedClusterArn),

			// Service selection with dynamic options based on cluster selection
			huh.NewSelect[string]().
				Title("Select Service").
				OptionsFunc(func() []huh.Option[string] {
					// Return empty options if no cluster selected yet
					if selectedClusterArn == "" {
						return []huh.Option[string]{}
					}

					// Get services for selected cluster
					listServices, err := client.ListServices(ctx, &ecs.ListServicesInput{
						Cluster: &selectedClusterArn,
					})
					if err != nil {
						// Just return empty options on error, the final validation will catch it
						return []huh.Option[string]{}
					}
					if len(listServices.ServiceArns) == 0 {
						return []huh.Option[string]{}
					}

					return huh.NewOptions(listServices.ServiceArns...)
				}, &selectedClusterArn).
				Value(&selectedServiceArn),

			// Task selection with dynamic options based on service selection
			huh.NewSelect[string]().
				Title("Select Task").
				OptionsFunc(func() []huh.Option[string] {
					// Return empty options if no service selected yet
					if selectedClusterArn == "" || selectedServiceArn == "" {
						return []huh.Option[string]{}
					}

					// Get tasks for selected service
					listTasks, err := client.ListTasks(ctx, &ecs.ListTasksInput{
						Cluster:     &selectedClusterArn,
						ServiceName: &selectedServiceArn,
					})
					if err != nil {
						return []huh.Option[string]{}
					}
					if len(listTasks.TaskArns) == 0 {
						return []huh.Option[string]{}
					}

					return huh.NewOptions(listTasks.TaskArns...)
				}, &selectedServiceArn).
				Value(&selectedTaskArn),

			// Container selection with dynamic options based on task selection
			huh.NewSelect[string]().
				Title("Select Container").
				OptionsFunc(func() []huh.Option[string] {
					// Return empty options if no task selected yet
					if selectedClusterArn == "" || selectedTaskArn == "" {
						return []huh.Option[string]{}
					}

					// Get task details to extract container names
					describeTasks, err := client.DescribeTasks(ctx, &ecs.DescribeTasksInput{
						Cluster: &selectedClusterArn,
						Tasks:   []string{selectedTaskArn},
					})
					if err != nil {
						return []huh.Option[string]{}
					}
					if len(describeTasks.Tasks) == 0 {
						return []huh.Option[string]{}
					}

					// We still need task details here to get container names
					// But we won't store the task in result yet
					task := &describeTasks.Tasks[0]

					// Build container options
					var containerNames []string
					for _, container := range task.Containers {
						containerNames = append(containerNames, *container.Name)
					}

					if len(containerNames) == 0 {
						return []huh.Option[string]{}
					}

					return huh.NewOptions(containerNames...)
				}, &selectedTaskArn).
				Value(&selectedContainerName),
		),
	)

	// Run the combined form
	err = form.Run()
	if err != nil {
		return nil, fmt.Errorf("form error: %w", err)
	}

	// After the form exits, explicitly describe each selected component to ensure we have complete data

	// Describe selected cluster
	describeClusters, err := client.DescribeClusters(ctx, &ecs.DescribeClustersInput{
		Clusters: []string{selectedClusterArn},
	})
	if err != nil {
		return nil, fmt.Errorf("failed to describe cluster after selection: %w", err)
	}
	if len(describeClusters.Clusters) == 0 {
		return nil, fmt.Errorf("cluster not found: %s", selectedClusterArn)
	}
	result.Cluster = &describeClusters.Clusters[0]

	// Describe selected service
	describeService, err := client.DescribeServices(ctx, &ecs.DescribeServicesInput{
		Cluster:  &selectedClusterArn,
		Services: []string{selectedServiceArn},
	})
	if err != nil {
		return nil, fmt.Errorf("failed to describe service after selection: %w", err)
	}
	if len(describeService.Services) == 0 {
		return nil, fmt.Errorf("service not found: %s", selectedServiceArn)
	}
	result.Service = &describeService.Services[0]

	// Describe selected task
	describeTasks, err := client.DescribeTasks(ctx, &ecs.DescribeTasksInput{
		Cluster: &selectedClusterArn,
		Tasks:   []string{selectedTaskArn},
	})
	if err != nil {
		return nil, fmt.Errorf("failed to describe task after selection: %w", err)
	}
	if len(describeTasks.Tasks) == 0 {
		return nil, fmt.Errorf("task not found: %s", selectedTaskArn)
	}
	result.Task = &describeTasks.Tasks[0]

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
