package selector

import (
	"context"
	"fmt"

	"github.com/aws/aws-sdk-go-v2/service/ecs"
	"github.com/aws/aws-sdk-go-v2/service/ecs/types"
	"github.com/pterm/pterm"
)

type Client interface {
	DescribeClusters(ctx context.Context, input *ecs.DescribeClustersInput, options ...func(*ecs.Options)) (*ecs.DescribeClustersOutput, error)
	DescribeServices(ctx context.Context, input *ecs.DescribeServicesInput, options ...func(*ecs.Options)) (*ecs.DescribeServicesOutput, error)
	DescribeTasks(ctx context.Context, input *ecs.DescribeTasksInput, options ...func(*ecs.Options)) (*ecs.DescribeTasksOutput, error)
	ListClusters(ctx context.Context, input *ecs.ListClustersInput, options ...func(*ecs.Options)) (*ecs.ListClustersOutput, error)
	ListServices(ctx context.Context, input *ecs.ListServicesInput, options ...func(*ecs.Options)) (*ecs.ListServicesOutput, error)
	ListTasks(ctx context.Context, input *ecs.ListTasksInput, options ...func(*ecs.Options)) (*ecs.ListTasksOutput, error)
}

type DefaultSelector struct{}

type Selector interface {
	Select(title string, options []string) (string, error)
}

func (s *DefaultSelector) Select(title string, options []string) (string, error) {
	return pterm.DefaultInteractiveSelect.WithOptions(options).Show(title)
}

func SelectCluster(ctx context.Context, client Client, selector Selector, clusterId string) (*types.Cluster, error) {
	if clusterId == "" {
		listClusters, err := client.ListClusters(ctx, &ecs.ListClustersInput{})
		if err != nil {
			return nil, err
		}
		clusterArn, err := selector.Select("Cluster", listClusters.ClusterArns)
		if err != nil {
			return nil, err
		}
		clusterId = clusterArn
	}
	describeClusters, err := client.DescribeClusters(ctx, &ecs.DescribeClustersInput{
		Clusters: []string{clusterId},
	})
	if err != nil {
		return nil, err
	}
	if len(describeClusters.Clusters) > 0 {
		return &describeClusters.Clusters[0], nil
	}
	return nil, fmt.Errorf("no cluster '%v' found", clusterId)
}

func SelectService(ctx context.Context, client Client, selector Selector, clusterId string, serviceId string) (*types.Service, error) {
	if serviceId == "" {
		listServices, err := client.ListServices(ctx, &ecs.ListServicesInput{
			Cluster: &clusterId,
		})
		if err != nil {
			return nil, err
		}
		serviceArn, err := selector.Select("Service", listServices.ServiceArns)
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

func SelectTask(ctx context.Context, client Client, clusterId string, serviceId string, taskId string) (*types.Task, error) {
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
