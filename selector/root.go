package selector

import (
	"context"
	"fmt"
	"log"
	"slices"
	"strconv"
	"strings"

	"github.com/aws/aws-sdk-go-v2/service/ecs/types"
	"github.com/charmbracelet/huh"
	"github.com/charmbracelet/lipgloss"
	"github.com/sestrella/iecs/client"
)

var titleStyle = lipgloss.NewStyle().Bold(true)

type Selectors interface {
	Cluster(ctx context.Context) (*types.Cluster, error)
	Service(ctx context.Context, cluster *types.Cluster) (*types.Service, error)
	Task(ctx context.Context, service *types.Service) (*types.Task, error)
	Tasks(ctx context.Context, service *types.Service) ([]types.Task, error)
	ServiceConfig(ctx context.Context, service *types.Service) (*client.ServiceConfig, error)
	Container(ctx context.Context, containers []types.Container) (*types.Container, error)
	ContainerDefinitions(
		ctx context.Context,
		taskDefinitionArn string,
	) ([]types.ContainerDefinition, error)
}

type ClientSelectors struct {
	client client.Client
	theme  huh.Theme
}

func NewSelectors(client client.Client, theme huh.Theme) Selectors {
	return ClientSelectors{client: client, theme: theme}
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
					Title("Select a cluster").
					Options(
						huh.NewOptions(clusterArns...)...,
					).
					Value(&selectedClusterArn).
					WithHeight(5),
			),
		).WithTheme(&cs.theme)
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
					Title("Select a cluster").
					Options(huh.NewOptions(serviceArns...)...).
					Value(&selectedServiceArn).
					WithHeight(5),
			),
		).WithTheme(&cs.theme)

		if err = form.Run(); err != nil {
			return nil, err
		}
	}

	services, err := cs.client.DescribeServices(
		ctx,
		*cluster.ClusterArn,
		[]string{selectedServiceArn},
	)
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
					Title("Select a task").
					Options(huh.NewOptions(taskArns...)...).
					Value(&selectedTaskArn).
					WithHeight(5),
			),
		).WithTheme(&cs.theme)
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
					Title("Select at least one task").
					Options(huh.NewOptions(taskArns...)...).
					Value(&selectedTaskArns).
					Validate(func(s []string) error {
						if len(s) > 0 {
							return nil
						}
						return fmt.Errorf("no task selected")
					}),
			),
		).WithTheme(&cs.theme)
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

func (cs ClientSelectors) ServiceConfig(
	ctx context.Context,
	service *types.Service,
) (*client.ServiceConfig, error) {
	currentTaskDefinition, err := cs.client.DescribeTaskDefinition(ctx, *service.TaskDefinition)
	if err != nil {
		return nil, err
	}

	taskDefinitionArns, err := cs.client.ListTaskDefinitions(ctx, *currentTaskDefinition.Family)
	if err != nil {
		return nil, err
	}
	if len(taskDefinitionArns) == 0 {
		return nil, fmt.Errorf("no task definitions")
	}

	var taskDefinitionArn = currentTaskDefinition.TaskDefinitionArn
	var desiredCountStr = strconv.FormatInt(int64(service.DesiredCount), 10)
	form := huh.NewForm(
		huh.NewGroup(
			huh.NewSelect[string]().
				Title("Task definition").
				Options(huh.NewOptions(taskDefinitionArns...)...).
				Value(taskDefinitionArn).
				WithHeight(5),
			huh.NewInput().
				Title("Desired count").
				Value(&desiredCountStr).
				Validate(func(s string) error {
					val, err := strconv.ParseInt(s, 10, 32)
					if err != nil {
						return fmt.Errorf("invalid number")
					}
					if val < 0 {
						return fmt.Errorf("must be greater or equal to 0")
					}
					return nil
				}),
		),
	).WithTheme(&cs.theme)
	if err := form.Run(); err != nil {
		return nil, err
	}

	desiredCount, err := strconv.ParseInt(desiredCountStr, 10, 32)
	if err != nil {
		return nil, err
	}

	return &client.ServiceConfig{
		TaskDefinitionArn: *taskDefinitionArn,
		DesiredCount:      int32(desiredCount),
	}, nil
}

func (cs ClientSelectors) Container(
	ctx context.Context,
	containers []types.Container,
) (*types.Container, error) {
	var containerNames []string
	for _, container := range containers {
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
					Title("Select a container").
					Options(huh.NewOptions(containerNames...)...).
					Value(&containerName).
					WithHeight(5),
			),
		).WithTheme(&cs.theme)
		if err := form.Run(); err != nil {
			return nil, err
		}
	}

	for _, container := range containers {
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
					Title("Select at least one container").
					Options(huh.NewOptions(containerNames...)...).
					Value(&selectedContainerNames).
					Validate(func(s []string) error {
						if len(s) > 0 {
							return nil
						}
						return fmt.Errorf("no container selected")
					}),
			),
		).WithTheme(&cs.theme)
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

	fmt.Printf(
		"%s %s\n",
		titleStyle.Render("Container(s):"),
		strings.Join(selectedContainerNames, ","),
	)
	return selectedContainers, nil
}
