package cmd

import (
	"context"
	"fmt"
	"os"

	"github.com/aws/aws-sdk-go-v2/service/ecs"
	"github.com/aws/aws-sdk-go-v2/service/ecs/types"
	"github.com/pterm/pterm"
	"github.com/spf13/cobra"
)

var rootCmd = &cobra.Command{
	Use:     "iecs",
	Short:   "An interactive CLI for ECS",
	Long:    "Performs commons tasks on ECS, such as getting remote access or viewing logs",
	Version: "0.1.0",
}

func Execute() {
	err := rootCmd.Execute()
	if err != nil {
		os.Exit(1)
	}
}

func describeCluster(ctx context.Context, client *ecs.Client, clusterId string) (*types.Cluster, error) {
	if clusterId == "" {
		listClusters, err := client.ListClusters(ctx, &ecs.ListClustersInput{})
		if err != nil {
			return nil, err
		}
		clusterArn, err := pterm.DefaultInteractiveSelect.WithOptions(listClusters.ClusterArns).Show("Cluster")
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
	if len(describeClusters.Clusters) == 1 {
		return &describeClusters.Clusters[0], nil
	}
	return nil, fmt.Errorf("no cluster '%v' found", clusterId)
}

func describeService(ctx context.Context, client *ecs.Client, clusterId string, serviceId string) (*types.Service, error) {
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
	return &describeService.Services[0], nil
}

func describeTask(ctx context.Context, client *ecs.Client, clusterId string, serviceId string, taskId string) (*types.Task, error) {
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
	if len(describeTasks.Tasks) == 1 {
		return &describeTasks.Tasks[0], nil
	}
	return nil, fmt.Errorf("no task '%v' found", taskId)
}
