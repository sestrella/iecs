package selector

import (
	"context"
	"fmt"

	"github.com/aws/aws-sdk-go-v2/service/ecs"
	"github.com/aws/aws-sdk-go-v2/service/ecs/types"
	"github.com/pterm/pterm"
)

func SelectService(ctx context.Context, client *ecs.Client, clusterId string, serviceId string) (*types.Service, error) {
	if serviceId == "" {
		listServices, err := client.ListServices(ctx, &ecs.ListServicesInput{
			Cluster: &clusterId,
		})
		if err != nil {
			return nil, err
		}
		serviceArn, err := pterm.DefaultInteractiveSelect.WithOptions(listServices.ServiceArns).Show("Service")
		if err != nil {
			return nil, err
		}
		serviceId = serviceArn
	}
	describeService, err := client.DescribeServices(ctx, &ecs.DescribeServicesInput{
		Cluster:  &clusterId,
		Services: []string{serviceId},
	})
	if err != nil {
		return nil, err
	}
	if len(describeService.Services) > 0 {
		return &describeService.Services[0], nil
	}
	return nil, fmt.Errorf("no service '%v' found", serviceId)
}

func SelectTask(ctx context.Context, client *ecs.Client, clusterId string, serviceId string, taskId string) (*types.Task, error) {
	if taskId == "" {
		listTasks, err := client.ListTasks(ctx, &ecs.ListTasksInput{
			Cluster:     &clusterId,
			ServiceName: &serviceId,
		})
		if err != nil {
			return nil, err
		}
		taskArn, err := pterm.DefaultInteractiveSelect.WithOptions(listTasks.TaskArns).Show("Task")
		if err != nil {
			return nil, err
		}
		taskId = taskArn
	}
	describeTasks, err := client.DescribeTasks(ctx, &ecs.DescribeTasksInput{
		Cluster: &clusterId,
		Tasks:   []string{taskId},
	})
	if err != nil {
		return nil, err
	}
	if len(describeTasks.Tasks) > 0 {
		return &describeTasks.Tasks[0], nil
	}
	return nil, fmt.Errorf("no task '%v' found", taskId)
}

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
