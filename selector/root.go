package selector

import (
	"context"

	"github.com/aws/aws-sdk-go-v2/service/ecs/types"
	"github.com/sestrella/iecs/client"
)

// SelectedContainer holds the selected cluster, service, task and container
type SelectedContainer struct {
	Cluster   *types.Cluster
	Service   *types.Service
	Task      *types.Task
	Container *types.Container
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
