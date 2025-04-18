package selector

import (
	"fmt"

	"github.com/aws/aws-sdk-go-v2/service/ecs/types"
	"github.com/charmbracelet/huh"
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
			fmt.Printf("Selected container: %s\n", *container.Name)
			return &container, nil
		}
	}

	return nil, fmt.Errorf("container not found: %s", selectedContainerName)
}
