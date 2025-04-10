package selector

import (
	"context"
	"fmt"

	"github.com/aws/aws-sdk-go-v2/service/ecs/types"
	"github.com/charmbracelet/huh"
	"github.com/sestrella/iecs/client/ecs"
)

// SelectedContainer holds the selected cluster, service, task and container
type SelectedContainer struct {
	Cluster   *types.Cluster
	Service   *types.Service
	Task      *types.Task
	Container *types.Container
}

// RunContainerSelector runs an interactive form to select an ECS cluster, service, task and container
func RunContainerSelector(
	ctx context.Context,
	client ecs.Client,
) (*SelectedContainer, error) {
	result := &SelectedContainer{}

	// Variables to store form selections
	var selectedClusterArn string
	var selectedServiceArn string
	var selectedTaskArn string
	var selectedContainerName string

	// Get clusters
	clusterArns, err := client.ListClusters(ctx)
	if err != nil {
		return nil, fmt.Errorf("failed to list clusters: %w", err)
	}

	// Create the form with all selections
	form := huh.NewForm(
		huh.NewGroup(
			huh.NewSelect[string]().
				Title("Select Cluster").
				Options(
					huh.NewOptions(clusterArns...)...,
				).
				Value(&selectedClusterArn).
				WithHeight(5),

			// Service selection with dynamic options based on cluster selection
			huh.NewSelect[string]().
				Title("Select Service").
				OptionsFunc(func() []huh.Option[string] {
					// Return empty options if no cluster selected yet
					if selectedClusterArn == "" {
						return []huh.Option[string]{}
					}

					// Get services for selected cluster
					serviceArns, err := client.ListServices(ctx, selectedClusterArn)
					if err != nil {
						// Just return empty options on error, the final validation will catch it
						return []huh.Option[string]{}
					}
					if len(serviceArns) == 0 {
						return []huh.Option[string]{}
					}

					return huh.NewOptions(serviceArns...)
				}, &selectedClusterArn).
				Value(&selectedServiceArn).
				WithHeight(5),
		),
		huh.NewGroup(
			// Task selection with dynamic options based on service selection
			huh.NewSelect[string]().
				Title("Select Task").
				OptionsFunc(func() []huh.Option[string] {
					// Return empty options if no service selected yet
					if selectedClusterArn == "" || selectedServiceArn == "" {
						return []huh.Option[string]{}
					}

					// Get tasks for selected service
					taskArns, err := client.ListTasks(ctx, selectedClusterArn, selectedServiceArn)
					if err != nil {
						return []huh.Option[string]{}
					}
					if len(taskArns) == 0 {
						return []huh.Option[string]{}
					}

					return huh.NewOptions(taskArns...)
				}, &selectedServiceArn).
				Value(&selectedTaskArn).
				WithHeight(5),

			// Container selection with dynamic options based on task selection
			huh.NewSelect[string]().
				Title("Select Container").
				OptionsFunc(func() []huh.Option[string] {
					// Return empty options if no task selected yet
					if selectedClusterArn == "" || selectedTaskArn == "" {
						return []huh.Option[string]{}
					}

					// Get task details to extract container names
					task, err := client.DescribeTask(ctx, selectedClusterArn, selectedTaskArn)
					if err != nil {
						return []huh.Option[string]{}
					}
					if task == nil {
						return []huh.Option[string]{}
					}

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
				Value(&selectedContainerName).
				WithHeight(5),
		),
	)

	// Run the combined form
	err = form.Run()
	if err != nil {
		return nil, fmt.Errorf("form error: %w", err)
	}

	// After the form exits, explicitly describe each selected component to ensure we have complete data

	// Describe selected cluster
	cluster, err := client.DescribeCluster(ctx, selectedClusterArn)
	if err != nil {
		return nil, fmt.Errorf("failed to describe cluster after selection: %w", err)
	}
	result.Cluster = cluster

	// Describe selected service
	service, err := client.DescribeService(ctx, selectedClusterArn, selectedServiceArn)
	if err != nil {
		return nil, fmt.Errorf("failed to describe service after selection: %w", err)
	}
	result.Service = service

	// Describe selected task
	task, err := client.DescribeTask(ctx, selectedClusterArn, selectedTaskArn)
	if err != nil {
		return nil, fmt.Errorf("failed to describe task after selection: %w", err)
	}
	result.Task = task

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
