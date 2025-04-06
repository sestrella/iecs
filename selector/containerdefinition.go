package selector

import (
	"context"
	"fmt"

	"github.com/aws/aws-sdk-go-v2/service/ecs"
	"github.com/aws/aws-sdk-go-v2/service/ecs/types"
	"github.com/charmbracelet/huh"
	"github.com/sestrella/iecs/client"
)

// SelectedContainer holds the selected cluster, service, task and container
type SelectedContainerDefinition struct {
	Cluster             *types.Cluster
	Service             *types.Service
	TaskDefinition      *types.TaskDefinition
	ContainerDefinition *types.ContainerDefinition
}

// RunContainerSelector runs an interactive form to select an ECS cluster, service, task and container
func RunContainerDefinitionSelector(
	ctx context.Context,
	client client.Client,
) (*SelectedContainerDefinition, error) {
	result := &SelectedContainerDefinition{}

	// Variables to store form selections
	var selectedClusterArn string
	var selectedServiceArn string
	var selectedContainerDefinitionName string

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
				Value(&selectedServiceArn).
				WithHeight(5),
		),
		huh.NewGroup(
			// Container selection with dynamic options based on task selection
			huh.NewSelect[string]().
				Title("Select Container Definition").
				OptionsFunc(func() []huh.Option[string] {
					// Return empty options if no task selected yet
					if selectedClusterArn == "" || selectedServiceArn == "" {
						return []huh.Option[string]{}
					}

					describeServices, err := client.DescribeServices(
						ctx,
						&ecs.DescribeServicesInput{
							Cluster:  &selectedClusterArn,
							Services: []string{selectedServiceArn},
						},
					)
					if err != nil {
						return []huh.Option[string]{}
					}
					if len(describeServices.Services) == 0 {
						return []huh.Option[string]{}
					}
					result.Service = &describeServices.Services[0]

					describeTaskDefinition, err := client.DescribeTaskDefinition(
						ctx,
						&ecs.DescribeTaskDefinitionInput{
							TaskDefinition: result.Service.TaskDefinition,
						},
					)
					if err != nil {
						return []huh.Option[string]{}
					}
					result.TaskDefinition = describeTaskDefinition.TaskDefinition

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

	for _, containerDefinition := range result.TaskDefinition.ContainerDefinitions {
		if containerDefinition.Name == &selectedContainerDefinitionName {
			result.ContainerDefinition = &containerDefinition
			break
		}
	}

	return result, nil
}
