package selector

import (
	"context"
	"fmt"

	"github.com/aws/aws-sdk-go-v2/service/ecs/types"
	"github.com/charmbracelet/huh"
	"github.com/sestrella/iecs/client"
)

func ContainerSelector(containers []types.Container) (*types.Container, error) {
	var containerNames []string
	for _, container := range containers {
		containerNames = append(containerNames, *container.Name)
	}

	var selectedContainerName string
	if len(containerNames) == 1 {
		selectedContainerName = containerNames[0]
	} else {
		form := huh.NewForm(
			huh.NewGroup(
				huh.NewSelect[string]().
					Title("Select container").
					Options(huh.NewOptions(containerNames...)...).
					Value(&selectedContainerName).
					WithHeight(5),
			),
		)

		if err := form.Run(); err != nil {
			return nil, err
		}
	}

	for _, container := range containers {
		if *container.Name == selectedContainerName {
			fmt.Printf("Container: %s\n", *container.Name)
			return &container, nil
		}
	}

	return nil, fmt.Errorf("container not found: %s", selectedContainerName)
}

func ContainerDefinitionSelector(
	ctx context.Context,
	client client.Client,
	taskDefinitionArn string,
) (*types.ContainerDefinition, error) {
	taskDefinition, err := client.DescribeTaskDefinition(
		ctx,
		taskDefinitionArn,
	)
	if err != nil {
		return nil, err
	}

	var containerDefinitionNames []string
	for _, containerDefinition := range taskDefinition.ContainerDefinitions {
		containerDefinitionNames = append(containerDefinitionNames, *containerDefinition.Name)
	}

	var selectedContainerDefinitionName string
	if len(containerDefinitionNames) == 1 {
		selectedContainerDefinitionName = containerDefinitionNames[0]
	} else {
		form := huh.NewForm(
			huh.NewGroup(
				huh.NewSelect[string]().
					Title("Select container definition").
					Options(huh.NewOptions(containerDefinitionNames...)...).
					Value(&selectedContainerDefinitionName).
					WithHeight(5),
			),
		)

		if err = form.Run(); err != nil {
			return nil, err
		}
	}

	for _, containerDefinition := range taskDefinition.ContainerDefinitions {
		if *containerDefinition.Name == selectedContainerDefinitionName {
			fmt.Printf("Container definition: %s\n", *containerDefinition.Name)
			return &containerDefinition, nil
		}
	}

	return nil, fmt.Errorf("container definition not found: %s", selectedContainerDefinitionName)
}
