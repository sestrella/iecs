package selector

import (
	"context"
	"fmt"

	"github.com/aws/aws-sdk-go-v2/service/ecs"
	"github.com/aws/aws-sdk-go-v2/service/ecs/types"
	"github.com/pterm/pterm"
)

func SelectContainer(containers []types.Container, containerId string) (*types.Container, error) {
	if containerId == "" {
		var containerNames []string
		for _, container := range containers {
			containerNames = append(containerNames, *container.Name)
		}
		containerName, err := pterm.DefaultInteractiveSelect.WithOptions(containerNames).Show("Container")
		if err != nil {
			return nil, err
		}
		containerId = containerName
	}
	for _, container := range containers {
		if *container.Name == containerId {
			return &container, nil
		}
	}
	return nil, fmt.Errorf("no container '%v' found", containerId)
}

func SelectContainerDefinition(ctx context.Context, client *ecs.Client, taskDefinitionArn string, containerId string) (*types.ContainerDefinition, error) {
	describeTaskDefinition, err := client.DescribeTaskDefinition(ctx, &ecs.DescribeTaskDefinitionInput{
		TaskDefinition: &taskDefinitionArn,
	})
	if err != nil {
		return nil, err
	}
	containerDefinitions := describeTaskDefinition.TaskDefinition.ContainerDefinitions
	if containerId == "" {
		var containerNames []string
		for _, containerDefinition := range containerDefinitions {
			containerNames = append(containerNames, *containerDefinition.Name)
		}
		containerName, err := pterm.DefaultInteractiveSelect.WithOptions(containerNames).Show("Container")
		if err != nil {
			return nil, err
		}
		containerId = containerName
	}
	for _, containerDefinition := range containerDefinitions {
		if *containerDefinition.Name == containerId {
			return &containerDefinition, nil
		}
	}
	return nil, fmt.Errorf("no container '%v' found", containerId)
}
