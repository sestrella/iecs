package selector

import (
	"context"
	"fmt"
	"log"

	"github.com/aws/aws-sdk-go-v2/service/ecs"
	"github.com/aws/aws-sdk-go-v2/service/ecs/types"
	"github.com/charmbracelet/huh"
	"github.com/charmbracelet/lipgloss"
	"github.com/sestrella/iecs/client"
)

var titleStyle = lipgloss.NewStyle().Bold(true)

// SelectedContainerDefinition holds the selected cluster, service, task definition and container definition
type SelectedContainerDefinition struct {
	Cluster             *types.Cluster
	Service             *types.Service
	TaskDefinition      *types.TaskDefinition
	ContainerDefinition *types.ContainerDefinition
}

type Selectors interface {
	Cluster(ctx context.Context) (*types.Cluster, error)
	Container(containers []types.Container) (*types.Container, error)
	ContainerDefinition(
		ctx context.Context,
		taskDefinition string,
	) (*types.ContainerDefinition, error)
	Service(ctx context.Context, clusterArn string) (*types.Service, error)
	Task(ctx context.Context, clusterArn string, serviceArn string) (*types.Task, error)
}

var _ Selectors = ClientSelectors{}

type ClientSelectors struct {
	client    client.Client
	ecsClient *ecs.Client
}

func NewSelectors(client client.Client, ecsClient *ecs.Client) Selectors {
	return ClientSelectors{client: client, ecsClient: ecsClient}
}

func (cs ClientSelectors) Cluster(ctx context.Context) (*types.Cluster, error) {
	listClustersOuput, err := cs.ecsClient.ListClusters(ctx, &ecs.ListClustersInput{})
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
	describeClustersOutput, err := cs.ecsClient.DescribeClusters(ctx, &ecs.DescribeClustersInput{
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

func (cs ClientSelectors) Container(
	containers []types.Container,
) (*types.Container, error) {
	var containerNames []string
	for _, container := range containers {
		containerNames = append(containerNames, *container.Name)
	}

	var selectedContainerName string
	if len(containerNames) == 1 {
		log.Printf("Pre-select the only available container")
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
			fmt.Printf("%s %s\n", titleStyle.Render("Container:"), *container.Name)
			return &container, nil
		}
	}

	return nil, fmt.Errorf("container not found: %s", selectedContainerName)
}

func (cs ClientSelectors) ContainerDefinition(
	ctx context.Context,
	taskDefinitionArn string,
) (*types.ContainerDefinition, error) {
	taskDefinition, err := cs.client.DescribeTaskDefinition(
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
		log.Printf("Pre-select the only available container definition")
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
			fmt.Printf(
				"%s %s\n",
				titleStyle.Render("Container definition:"),
				*containerDefinition.Name,
			)
			return &containerDefinition, nil
		}
	}

	return nil, fmt.Errorf("container definition not found: %s", selectedContainerDefinitionName)
}

func (cs ClientSelectors) Service(ctx context.Context, clusterArn string) (*types.Service, error) {
	listServicesOutput, err := cs.ecsClient.ListServices(ctx, &ecs.ListServicesInput{
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
	output, err := cs.ecsClient.DescribeServices(ctx, &ecs.DescribeServicesInput{
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

func (cs ClientSelectors) Task(
	ctx context.Context,
	clusterArn string,
	serviceArn string,
) (*types.Task, error) {
	taskArns, err := cs.client.ListTasks(ctx, clusterArn, serviceArn)
	if err != nil {
		return nil, err
	}

	var selectedTaskArn string
	if len(taskArns) == 1 {
		log.Printf("Pre-select the only available task")
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

	task, err := cs.client.DescribeTask(ctx, clusterArn, selectedTaskArn)
	if err != nil {
		return nil, err
	}

	fmt.Printf("%s %s\n", titleStyle.Render("Task:"), *task.TaskArn)
	return task, nil
}
