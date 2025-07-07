package selector

import (
	"context"
	"fmt"
	"log"
	"slices"
	"strings"

	"github.com/aws/aws-sdk-go-v2/service/ecs/types"
	"github.com/charmbracelet/huh"
	"github.com/charmbracelet/lipgloss"
	"github.com/sestrella/iecs/client"
)

var (
	_          Selectors = ClientSelectors{}
	titleStyle           = lipgloss.NewStyle().Bold(true)
)

type Selectors interface {
	Cluster(ctx context.Context) (*types.Cluster, error)
	Service(ctx context.Context, cluster *types.Cluster) (*types.Service, error)
	Task(ctx context.Context, service *types.Service) (*types.Task, error)
	Tasks(ctx context.Context, service *types.Service) ([]types.Task, error)
	Container(ctx context.Context, task *types.Task) (*types.Container, error)
	ContainerDefinitions(
		ctx context.Context,
		taskDefinitionArn string,
	) ([]types.ContainerDefinition, error)
}

type ClientSelectors struct {
	client client.Client
}

func NewSelectors(client client.Client) Selectors {
	return ClientSelectors{client: client}
}

func (cs ClientSelectors) Cluster(ctx context.Context) (*types.Cluster, error) {
	clusterArns, err := cs.client.ListClusters(ctx)
	if err != nil {
		return nil, err
	}

	var selectedClusterArn string
	if len(clusterArns) == 1 {
		log.Printf("Pre-selecting the only available cluster")
		selectedClusterArn = clusterArns[0]
	} else {
		form := huh.NewForm(
			huh.NewGroup(
				huh.NewSelect[string]().
					Title("Cluster").
					Options(
						huh.NewOptions(clusterArns...)...,
					).
					Value(&selectedClusterArn).
					WithHeight(5),
			),
		)
		if err = form.Run(); err != nil {
			return nil, err
		}
	}

	clusters, err := cs.client.DescribeClusters(ctx, []string{selectedClusterArn})
	if err != nil {
		return nil, err
	}

	if len(clusters) == 0 {
		return nil, fmt.Errorf("cluster not found: %s", selectedClusterArn)
	}

	cluster := clusters[0]
	fmt.Printf("%s %s\n", titleStyle.Render("Cluster:"), *cluster.ClusterArn)
	return &cluster, nil
}

func (cs ClientSelectors) Service(
	ctx context.Context,
	cluster *types.Cluster,
) (*types.Service, error) {
	serviceArns, err := cs.client.ListServices(ctx, *cluster.ClusterArn)
	if err != nil {
		return nil, err
	}

	var selectedServiceArn string
	if len(serviceArns) == 1 {
		log.Printf("Pre-selecting the only available service")
		selectedServiceArn = serviceArns[0]
	} else {
		form := huh.NewForm(
			huh.NewGroup(
				huh.NewSelect[string]().
					Title("Service").
					Options(huh.NewOptions(serviceArns...)...).
					Value(&selectedServiceArn).
					WithHeight(5),
			),
		)

		if err = form.Run(); err != nil {
			return nil, err
		}
	}

	services, err := cs.client.DescribeServices(ctx, *cluster.ClusterArn, []string{selectedServiceArn})
	if err != nil {
		return nil, err
	}

	if len(services) == 0 {
		return nil, fmt.Errorf("service not found: %s", selectedServiceArn)
	}

	service := services[0]
	fmt.Printf("%s %s\n", titleStyle.Render("Service:"), *service.ServiceArn)
	return &service, nil
}

func (cs ClientSelectors) Task(
	ctx context.Context,
	service *types.Service,
) (*types.Task, error) {
	taskArns, err := cs.client.ListTasks(ctx, *service.ClusterArn, *service.ServiceArn)
	if err != nil {
		return nil, err
	}

	var selectedTaskArn string
	if len(taskArns) == 1 {
		log.Printf("Pre-selecting the only available task")
		selectedTaskArn = taskArns[0]
	} else {
		form := huh.NewForm(
			huh.NewGroup(
				huh.NewSelect[string]().
					Title("Select task").
					Options(huh.NewOptions(taskArns...)...).
					Value(&selectedTaskArn).
					WithHeight(5),
			),
		)
		if err = form.Run(); err != nil {
			return nil, err
		}
	}

	tasks, err := cs.client.DescribeTasks(ctx, *service.ClusterArn, []string{selectedTaskArn})
	if err != nil {
		return nil, err
	}

	if len(tasks) == 0 {
		return nil, fmt.Errorf("task not found: %s", selectedTaskArn)
	}

	task := tasks[0]
	fmt.Printf("%s %s\n", titleStyle.Render("Task:"), *task.TaskArn)
	return &task, nil
}

func (cs ClientSelectors) Tasks(
	ctx context.Context,
	service *types.Service,
) ([]types.Task, error) {
	taskArns, err := cs.client.ListTasks(ctx, *service.ClusterArn, *service.ServiceArn)
	if err != nil {
		return nil, err
	}

	var selectedTaskArns []string
	if len(taskArns) == 1 {
		log.Println("Pre-selecting the only task available")
		selectedTaskArns = append(selectedTaskArns, taskArns[0])
	} else {
		form := huh.NewForm(
			huh.NewGroup(
				huh.NewMultiSelect[string]().
					Title("Task(s)").
					Options(huh.NewOptions(taskArns...)...).
					Value(&selectedTaskArns).
					Validate(func(s []string) error {
						if len(s) > 0 {
							return nil
						}
						return fmt.Errorf("select at least one task")
					}),
			),
		)
		if err = form.Run(); err != nil {
			return nil, err
		}
	}

	fmt.Printf("%s %s\n", titleStyle.Render("Task(s):"), strings.Join(selectedTaskArns, ","))
	tasks, err := cs.client.DescribeTasks(ctx, *service.ClusterArn, selectedTaskArns)
	if err != nil {
		return nil, err
	}

	if len(tasks) == 0 {
		return nil, fmt.Errorf("no tasks selected")
	}

	return tasks, nil
}

func (cs ClientSelectors) Container(
	ctx context.Context,
	task *types.Task,
) (*types.Container, error) {
	var containerNames []string
	for _, container := range task.Containers {
		containerNames = append(containerNames, *container.Name)
	}

	var containerName string
	if len(containerNames) == 1 {
		log.Printf("Pre-selecting the only available container")
		containerName = containerNames[0]
	} else {
		form := huh.NewForm(
			huh.NewGroup(
				huh.NewSelect[string]().
					Title("Select container").
					Options(huh.NewOptions(containerNames...)...).
					Value(&containerName).
					WithHeight(5),
			),
		)
		if err := form.Run(); err != nil {
			return nil, err
		}
	}

	for _, container := range task.Containers {
		if *container.Name == containerName {
			fmt.Printf("%s %s\n", titleStyle.Render("Container:"), *container.Name)
			return &container, nil
		}
	}

	return nil, fmt.Errorf("container not found: %s", containerName)
}

func (cs ClientSelectors) ContainerDefinitions(
	ctx context.Context,
	taskDefinitionArn string,
) ([]types.ContainerDefinition, error) {
	taskDefinition, err := cs.client.DescribeTaskDefinition(ctx, taskDefinitionArn)
	if err != nil {
		return nil, err
	}

	var containerNames []string
	for _, containerDefinition := range taskDefinition.ContainerDefinitions {
		containerNames = append(containerNames, *containerDefinition.Name)
	}

	var selectedContainerNames []string
	if len(containerNames) == 1 {
		log.Printf("Pre-selecting the only available container")
		selectedContainerNames = append(selectedContainerNames, containerNames[0])
	} else {
		form := huh.NewForm(
			huh.NewGroup(
				huh.NewMultiSelect[string]().
					Title("Containers").
					Options(huh.NewOptions(containerNames...)...).
					Value(&selectedContainerNames).
					Validate(func(s []string) error {
						if len(s) > 0 {
							return nil
						}
						return fmt.Errorf("select at least one container")
					}),
			),
		)
		if err = form.Run(); err != nil {
			return nil, err
		}
	}

	var selectedContainers []types.ContainerDefinition
	for _, containerDefinition := range taskDefinition.ContainerDefinitions {
		if slices.Contains(selectedContainerNames, *containerDefinition.Name) {
			selectedContainers = append(selectedContainers, containerDefinition)
		}
	}
	if len(selectedContainers) == 0 {
		return nil, fmt.Errorf("no containers selected")
	}

	return selectedContainers, nil
}
