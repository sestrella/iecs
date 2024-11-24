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

const (
	CLUSTER_FLAG   = "cluster"
	SERVICE_FLAG   = "service"
	TASK_FLAG      = "task"
	CONTAINER_FLAG = "container"
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

func selectCluster(ctx context.Context, client *ecs.Client, clusterId string) (*types.Cluster, error) {
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
	if len(describeClusters.Clusters) > 0 {
		return &describeClusters.Clusters[0], nil
	}
	return nil, fmt.Errorf("no cluster '%v' found", clusterId)
}

func selectService(ctx context.Context, client *ecs.Client, clusterId string, serviceId string) (*types.Service, error) {
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

func selectTask(ctx context.Context, client *ecs.Client, clusterId string, serviceId string, taskId string) (*types.Task, error) {
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

func init() {
	rootCmd.PersistentFlags().StringP(CLUSTER_FLAG, "l", "", "cluster id or ARN")
	rootCmd.PersistentFlags().StringP(SERVICE_FLAG, "s", "", "service id or ARN")
	rootCmd.PersistentFlags().StringP(TASK_FLAG, "t", "", "task id or ARN")
	rootCmd.PersistentFlags().StringP(CONTAINER_FLAG, "n", "", "container name")
}
