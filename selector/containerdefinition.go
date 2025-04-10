package selector

import (
	"context"
	"fmt"

	"github.com/aws/aws-sdk-go-v2/service/ecs/types"
	"github.com/charmbracelet/huh"
	"github.com/sestrella/iecs/client"
)

// SelectedContainerDefinition holds the selected cluster, service, task definition and container definition
type SelectedContainerDefinition struct {
	Cluster             *types.Cluster
	Service             *types.Service
	TaskDefinition      *types.TaskDefinition
	ContainerDefinition *types.ContainerDefinition
}

// RunContainerDefinitionSelector runs an interactive form to select an ECS cluster, service and container definition
func RunContainerDefinitionSelector(
	ctx context.Context,
	client client.ClientV2,
) (*SelectedContainerDefinition, error) {
	result := &SelectedContainerDefinition{}

	// Variables to store form selections
	var selectedClusterArn string
	var selectedServiceArn string
	var selectedContainerDefinitionName string

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
			// Container selection with dynamic options based on service selection
			huh.NewSelect[string]().
				Title("Select Container Definition").
				OptionsFunc(func() []huh.Option[string] {
					// Return empty options if no service selected yet
					if selectedClusterArn == "" || selectedServiceArn == "" {
						return []huh.Option[string]{}
					}

					service, err := client.DescribeService(
						ctx,
						selectedClusterArn,
						selectedServiceArn,
					)
					if err != nil {
						return []huh.Option[string]{}
					}
					if service == nil {
						return []huh.Option[string]{}
					}
					result.Service = service

					taskDefinition, err := client.DescribeTaskDefinition(
						ctx,
						*service.TaskDefinition,
					)
					if err != nil {
						return []huh.Option[string]{}
					}
					if taskDefinition == nil {
						return []huh.Option[string]{}
					}
					result.TaskDefinition = taskDefinition

					var containerNames []string
					for _, containerDefinition := range result.TaskDefinition.ContainerDefinitions {
						containerNames = append(containerNames, *containerDefinition.Name)
					}
					if len(containerNames) == 0 {
						return []huh.Option[string]{}
					}

					return huh.NewOptions(containerNames...)
				}, &selectedServiceArn).
				Value(&selectedContainerDefinitionName).
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

	// Find selected container definition
	for _, containerDefinition := range result.TaskDefinition.ContainerDefinitions {
		if *containerDefinition.Name == selectedContainerDefinitionName {
			result.ContainerDefinition = &containerDefinition
			break
		}
	}

	if result.ContainerDefinition == nil {
		return nil, fmt.Errorf(
			"container definition not found: %s",
			selectedContainerDefinitionName,
		)
	}

	return result, nil
}
