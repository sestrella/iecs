package selector

import (
	"context"

	"github.com/aws/aws-sdk-go-v2/service/ecs/types"
	"github.com/charmbracelet/lipgloss"
	"github.com/sestrella/iecs/client"
)

var titleStyle = lipgloss.NewStyle().Bold(true)

// SelectedContainer holds the selected cluster, service, task and container
type SelectedContainer struct {
	Cluster   *types.Cluster
	Service   *types.Service
	Task      *types.Task
	Container *types.Container
}

// SelectedContainerDefinition holds the selected cluster, service, task definition and container definition
type SelectedContainerDefinition struct {
	Cluster             *types.Cluster
	Service             *types.Service
	TaskDefinition      *types.TaskDefinition
	ContainerDefinition *types.ContainerDefinition
}

type Selector interface {
	Cluster(ctx context.Context) (*types.Cluster, error)
	Container(containers []types.Container) (*types.Container, error)
	ContainerDefinition(
		ctx context.Context,
		taskDefinition string,
	) (*types.ContainerDefinition, error)
	Service(ctx context.Context, clusterArn string) (*types.Service, error)
	Task(ctx context.Context, clusterArn string, serviceArn string) (*types.Task, error)
}

var _ Selector = ClientSelector{}

type ClientSelector struct {
	client client.Client
}

func (cs ClientSelector) Cluster(ctx context.Context) (*types.Cluster, error) {
	return ClusterSelector(ctx, cs.client)
}

func (cs ClientSelector) Container(
	containers []types.Container,
) (*types.Container, error) {
	return ContainerSelector(containers)
}

func (cs ClientSelector) ContainerDefinition(
	ctx context.Context,
	taskDefinition string,
) (*types.ContainerDefinition, error) {
	return ContainerDefinitionSelector(ctx, cs.client, taskDefinition)
}

func (cs ClientSelector) Service(ctx context.Context, clusterArn string) (*types.Service, error) {
	return ServiceSelector(ctx, cs.client, clusterArn)
}

func (cs ClientSelector) Task(
	ctx context.Context,
	clusterArn string,
	serviceArn string,
) (*types.Task, error) {
	return TaskSelector(ctx, cs.client, clusterArn, serviceArn)
}

func RunContainerSelector(
	ctx context.Context,
	client client.Client,
) (*SelectedContainer, error) {
	cluster, err := ClusterSelector(ctx, client)
	if err != nil {
		return nil, err
	}

	service, err := ServiceSelector(ctx, client, *cluster.ClusterArn)
	if err != nil {
		return nil, err
	}

	task, err := TaskSelector(ctx, client, *cluster.ClusterArn, *service.ServiceArn)
	if err != nil {
		return nil, err
	}

	container, err := ContainerSelector(task.Containers)
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

// RunContainerDefinitionSelector runs an interactive form to select an ECS cluster, service and container definition
func RunContainerDefinitionSelector(
	ctx context.Context,
	client client.Client,
) (*SelectedContainerDefinition, error) {
	cluster, err := ClusterSelector(ctx, client)
	if err != nil {
		return nil, err
	}

	service, err := ServiceSelector(ctx, client, *cluster.ClusterArn)
	if err != nil {
		return nil, err
	}

	containerDefinition, err := ContainerDefinitionSelector(ctx, client, *service.TaskDefinition)
	if err != nil {
		return nil, err
	}

	return &SelectedContainerDefinition{
		Cluster:             cluster,
		Service:             service,
		ContainerDefinition: containerDefinition,
	}, nil
}
