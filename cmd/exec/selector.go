package exec

import (
	"context"
	"fmt"

	"github.com/aws/aws-sdk-go-v2/service/ecs/types"
	"github.com/charmbracelet/huh"
	"github.com/sestrella/iecs/client"
)

type Selector struct {
	cluster   *types.Cluster
	service   *types.Service
	task      *types.Task
	container *types.Container
}

func newSelector(ctx context.Context, client client.Client) (*Selector, error) {
	clusterArns, err := client.ListClusters(ctx)
	if err != nil {
		return nil, err
	}

	var clusterArn string
	var serviceArn string
	var taskArn string
	var containerName string
	form := huh.NewForm(
		huh.NewGroup(
			huh.NewSelect[string]().
				Title("Cluster").
				Options(huh.NewOptions(clusterArns...)...).
				Value(&clusterArn),
			huh.NewSelect[string]().
				Title("Service").
				OptionsFunc(func() []huh.Option[string] {
					serviceArns, err := client.ListServices(ctx, clusterArn)
					if err != nil {
						return nil
					}

					return huh.NewOptions(serviceArns...)
				}, &clusterArn).
				Value(&serviceArn),
		),
		huh.NewGroup(
			huh.NewSelect[string]().
				Title("Task").
				OptionsFunc(func() []huh.Option[string] {
					taskArns, err := client.ListTasks(ctx, clusterArn, serviceArn)
					if err != nil {
						return nil
					}

					return huh.NewOptions(taskArns...)
				}, &serviceArn).
				Value(&taskArn),
			huh.NewSelect[string]().
				Title("Container").
				OptionsFunc(func() []huh.Option[string] {
					tasks, err := client.DescribeTasks(ctx, clusterArn, []string{taskArn})
					if err != nil {
						return nil
					}

					task := tasks[0]

					var containerNames []string
					for _, container := range task.Containers {
						containerNames = append(containerNames, *container.Name)
					}

					return huh.NewOptions(containerNames...)
				}, &taskArn).
				Value(&containerName),
		),
	)

	if err := form.Run(); err != nil {
		return nil, err
	}

	fmt.Printf("Cluster: %s\n", clusterArn)
	fmt.Printf("Service: %s\n", serviceArn)
	fmt.Printf("Task: %s\n", taskArn)
	fmt.Printf("Container: %s\n", containerName)

	clusters, err := client.DescribeClusters(ctx, []string{clusterArn})
	if err != nil {
		return nil, err
	}

	cluster := clusters[0]

	services, err := client.DescribeServices(ctx, clusterArn, []string{serviceArn})
	if err != nil {
		return nil, err
	}

	service := services[0]

	tasks, err := client.DescribeTasks(ctx, clusterArn, []string{taskArn})
	if err != nil {
		return nil, err
	}

	task := tasks[0]

	var selectedContainer types.Container
	for _, container := range task.Containers {
		if *container.Name == containerName {
			selectedContainer = container
			break
		}
	}

	return &Selector{
		cluster:   &cluster,
		service:   &service,
		task:      &task,
		container: &selectedContainer,
	}, nil
}
