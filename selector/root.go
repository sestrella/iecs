package selector

import (
	"context"
	"fmt"
	"log"

	"github.com/aws/aws-sdk-go-v2/service/ecs"
	"github.com/aws/aws-sdk-go-v2/service/ecs/types"
	"github.com/charmbracelet/huh"
	"github.com/charmbracelet/lipgloss"
)

var (
	_          Selectors = ClientSelectors{}
	titleStyle           = lipgloss.NewStyle().Bold(true)
)

type SelectedContainer struct {
	Cluster   *types.Cluster
	Service   *types.Service
	Task      *types.Task
	Container *types.Container
}

type SelectedContainerDefinition struct {
	Cluster             *types.Cluster
	Service             *types.Service
	ContainerDefinition *types.ContainerDefinition
}

type Selectors interface {
	RunContainerSelector(ctx context.Context) (*SelectedContainer, error)
	RunContainerDefinitionSelector(ctx context.Context) (*SelectedContainerDefinition, error)
}

type ClientSelectors struct {
	client *ecs.Client
}

func NewSelectors(client *ecs.Client) Selectors {
	return ClientSelectors{client: client}
}

func (cs ClientSelectors) RunContainerSelector(
	ctx context.Context,
) (*SelectedContainer, error) {
	cluster, err := cluster(ctx, cs.client)
	if err != nil {
		return nil, err
	}

	service, err := service(ctx, cs.client, *cluster.ClusterArn)
	if err != nil {
		return nil, err
	}

	task, err := task(ctx, cs.client, *cluster.ClusterArn, *service.ServiceArn)
	if err != nil {
		return nil, err
	}

	container, err := container(task.Containers)
	if err != nil {
		return nil, err
	}

	return &SelectedContainer{
		Cluster:   cluster,
		Service:   service,
		Task:      task,
		Container: container,
	}, nil
}

func (cs ClientSelectors) RunContainerDefinitionSelector(
	ctx context.Context,
) (*SelectedContainerDefinition, error) {
	cluster, err := cluster(ctx, cs.client)
	if err != nil {
		return nil, err
	}

	service, err := service(ctx, cs.client, *cluster.ClusterArn)
	if err != nil {
		return nil, err
	}

	containerDefinition, err := containerDefinition(ctx, cs.client, *service.TaskDefinition)
	if err != nil {
		return nil, err
	}

	return &SelectedContainerDefinition{
		Cluster:             cluster,
		Service:             service,
		ContainerDefinition: containerDefinition,
	}, nil
}

func cluster(ctx context.Context, client *ecs.Client) (*types.Cluster, error) {
	listClustersOuput, err := client.ListClusters(ctx, &ecs.ListClustersInput{})
	if err != nil {
		return nil, err
	}
	clusterArns := listClustersOuput.ClusterArns
	if len(clusterArns) == 0 {
		return nil, fmt.Errorf("no clusters found")
	}
	var clusterArn string
	if len(clusterArns) == 1 {
		log.Printf("Pre-select the only available cluster")
		clusterArn = clusterArns[0]
	} else {
		form := huh.NewForm(
			huh.NewGroup(
				huh.NewSelect[string]().
					Title("Select cluster").
					Options(
						huh.NewOptions(clusterArns...)...,
					).
					Value(&clusterArn).
					WithHeight(5),
			),
		)
		if err = form.Run(); err != nil {
			return nil, err
		}
	}
	describeClustersOutput, err := client.DescribeClusters(ctx, &ecs.DescribeClustersInput{
		Clusters: []string{clusterArn},
	})
	if err != nil {
		return nil, err
	}
	clusters := describeClustersOutput.Clusters
	if len(clusters) == 0 {
		return nil, fmt.Errorf("cluster not found: %s", clusterArn)
	}
	cluster := clusters[0]
	fmt.Printf("%s %s\n", titleStyle.Render("Cluster:"), *cluster.ClusterArn)
	return &cluster, nil
}

func container(
	containers []types.Container,
) (*types.Container, error) {
	var containerNames []string
	for _, container := range containers {
		containerNames = append(containerNames, *container.Name)
	}
	var containerName string
	if len(containerNames) == 1 {
		log.Printf("Pre-select the only available container")
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
	for _, container := range containers {
		if *container.Name == containerName {
			fmt.Printf("%s %s\n", titleStyle.Render("Container:"), *container.Name)
			return &container, nil
		}
	}
	return nil, fmt.Errorf("container not found: %s", containerName)
}

func containerDefinition(
	ctx context.Context,
	client *ecs.Client,
	taskDefinitionArn string,
) (*types.ContainerDefinition, error) {
	describeTaskDefinitionOutput, err := client.DescribeTaskDefinition(
		ctx,
		&ecs.DescribeTaskDefinitionInput{
			TaskDefinition: &taskDefinitionArn,
		},
	)
	if err != nil {
		return nil, err
	}
	taskDefinition := describeTaskDefinitionOutput.TaskDefinition
	var containerDefinitionNames []string
	for _, containerDefinition := range taskDefinition.ContainerDefinitions {
		containerDefinitionNames = append(containerDefinitionNames, *containerDefinition.Name)
	}
	var containerDefinitionName string
	if len(containerDefinitionNames) == 1 {
		log.Printf("Pre-select the only available container definition")
		containerDefinitionName = containerDefinitionNames[0]
	} else {
		form := huh.NewForm(
			huh.NewGroup(
				huh.NewSelect[string]().
					Title("Select container definition").
					Options(huh.NewOptions(containerDefinitionNames...)...).
					Value(&containerDefinitionName).
					WithHeight(5),
			),
		)
		if err = form.Run(); err != nil {
			return nil, err
		}
	}
	for _, containerDefinition := range taskDefinition.ContainerDefinitions {
		if *containerDefinition.Name == containerDefinitionName {
			fmt.Printf(
				"%s %s\n",
				titleStyle.Render("Container definition:"),
				*containerDefinition.Name,
			)
			return &containerDefinition, nil
		}
	}
	return nil, fmt.Errorf("container definition not found: %s", containerDefinitionName)
}

func service(ctx context.Context, client *ecs.Client, clusterArn string) (*types.Service, error) {
	listServicesOutput, err := client.ListServices(ctx, &ecs.ListServicesInput{
		Cluster: &clusterArn,
	})
	if err != nil {
		return nil, err
	}
	serviceArns := listServicesOutput.ServiceArns
	if len(serviceArns) == 0 {
		return nil, fmt.Errorf("no services found in cluster: %s", clusterArn)
	}
	var serviceArn string
	if len(serviceArns) == 1 {
		log.Printf("Pre-select the only available service")
		serviceArn = serviceArns[0]
	} else {
		form := huh.NewForm(
			huh.NewGroup(
				huh.NewSelect[string]().
					Title("Select service").
					Options(huh.NewOptions(serviceArns...)...).
					Value(&serviceArn).
					WithHeight(5),
			),
		)

		if err = form.Run(); err != nil {
			return nil, err
		}
	}
	output, err := client.DescribeServices(ctx, &ecs.DescribeServicesInput{
		Cluster:  &clusterArn,
		Services: []string{serviceArn},
	})
	if err != nil {
		return nil, err
	}
	services := output.Services
	if len(services) == 0 {
		return nil, fmt.Errorf("service not found: %s", serviceArn)
	}
	service := services[0]
	fmt.Printf("%s %s\n", titleStyle.Render("Service:"), *service.ServiceArn)
	return &service, nil
}

func task(
	ctx context.Context,
	client *ecs.Client,
	clusterArn string,
	serviceArn string,
) (*types.Task, error) {
	listTasksOutput, err := client.ListTasks(ctx, &ecs.ListTasksInput{
		Cluster:     &clusterArn,
		ServiceName: &serviceArn,
	})
	if err != nil {
		return nil, err
	}
	taskArns := listTasksOutput.TaskArns
	if len(taskArns) == 0 {
		return nil, fmt.Errorf(
			"no tasks found for service: %s in cluster: %s",
			serviceArn,
			clusterArn,
		)
	}
	var taskArn string
	if len(taskArns) == 1 {
		log.Printf("Pre-select the only available task")
		taskArn = taskArns[0]
	} else {
		form := huh.NewForm(
			huh.NewGroup(
				huh.NewSelect[string]().
					Title("Select task").
					Options(huh.NewOptions(taskArns...)...).
					Value(&taskArn).
					WithHeight(5),
			),
		)
		if err = form.Run(); err != nil {
			return nil, err
		}
	}
	describeTasksOutput, err := client.DescribeTasks(ctx, &ecs.DescribeTasksInput{
		Cluster: &clusterArn,
		Tasks:   []string{taskArn},
	})
	if err != nil {
		return nil, err
	}
	tasks := describeTasksOutput.Tasks
	if len(tasks) == 0 {
		return nil, fmt.Errorf("task not found: %s", taskArn)
	}
	task := tasks[0]
	fmt.Printf("%s %s\n", titleStyle.Render("Task:"), *task.TaskArn)
	return &task, nil
}
