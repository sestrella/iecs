package selector

import (
	"fmt"

	"github.com/aws/aws-sdk-go-v2/service/ecs/types"
	"github.com/pterm/pterm"
)

func SelectContainer(task types.Task, containerId string) (*types.Container, error) {
	var containerNames []string
	for _, container := range task.Containers {
		if *container.Name == containerId {
			return &container, nil
		}
		containerNames = append(containerNames, *container.Name)
	}
	containerName, err := pterm.DefaultInteractiveSelect.WithOptions(containerNames).Show("Select a container")
	if err != nil {
		return nil, fmt.Errorf("Error selecting a container: %w", err)
	}
	for _, container := range task.Containers {
		if *container.Name == containerName {
			return &container, nil
		}
	}
	return nil, fmt.Errorf("No container '%s' found in task '%s'", containerName, *task.TaskArn)
}
