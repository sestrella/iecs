/*
Copyright © 2024 NAME HERE <EMAIL ADDRESS>
*/
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

// rootCmd represents the base command when called without any subcommands
var rootCmd = &cobra.Command{
	Use:   "iecs",
	Short: "A brief description of your application",
	Long: `A longer description that spans multiple lines and likely contains
examples and usage of using your application. For example:

Cobra is a CLI library for Go that empowers applications.
This application is a tool to generate the needed files
to quickly create a Cobra application.`,
	// Uncomment the following line if your bare application
	// has an action associated with it:
	// Run: func(cmd *cobra.Command, args []string) { },
}

// Execute adds all child commands to the root command and sets flags appropriately.
// This is called by main.main(). It only needs to happen once to the rootCmd.
func Execute() {
	err := rootCmd.Execute()
	if err != nil {
		os.Exit(1)
	}
}

func describeCluster(ctx context.Context, client *ecs.Client, clusterId string) (*types.Cluster, error) {
	selectedClusterId, err := selectClusterId(ctx, client, clusterId)
	if err != nil {
		return nil, err
	}
	describedClusters, err := client.DescribeClusters(ctx, &ecs.DescribeClustersInput{
		Clusters: []string{*selectedClusterId},
	})
	if err != nil {
		return nil, err
	}
	if len(describedClusters.Clusters) == 1 {
		return &describedClusters.Clusters[0], nil
	}
	return nil, fmt.Errorf("no cluster '%v' found", *selectedClusterId)
}

func selectClusterId(ctx context.Context, client *ecs.Client, clusterId string) (*string, error) {
	if clusterId == "" {
		listedClusters, err := client.ListClusters(ctx, &ecs.ListClustersInput{})
		if err != nil {
			return nil, err
		}
		clusterArn, err := pterm.DefaultInteractiveSelect.WithOptions(listedClusters.ClusterArns).Show("Cluster")
		if err != nil {
			return nil, err
		}
		return &clusterArn, nil
	}
	return &clusterId, nil
}

func describeTask(ctx context.Context, client *ecs.Client, clusterId string, taskId string) (*types.Task, error) {
	selectedTaskId, err := selectTaskId(ctx, client, clusterId, taskId)
	if err != nil {
		return nil, err
	}
	describedTasks, err := client.DescribeTasks(ctx, &ecs.DescribeTasksInput{
		Cluster: &clusterId,
		Tasks:   []string{*selectedTaskId},
	})
	if err != nil {
		return nil, err
	}
	if len(describedTasks.Tasks) == 1 {
		return &describedTasks.Tasks[0], nil
	}
	return nil, fmt.Errorf("no task '%v' found", selectedTaskId)
}

func selectTaskId(ctx context.Context, client *ecs.Client, clusterId string, taskId string) (*string, error) {
	if taskId == "" {
		listTasks, _ := client.ListTasks(ctx, &ecs.ListTasksInput{
			Cluster: &clusterId,
		})
		taskArn, err := pterm.DefaultInteractiveSelect.WithOptions(listTasks.TaskArns).Show("Task")
		if err != nil {
			return nil, err
		}
		return &taskArn, nil
	}
	return &taskId, nil
}

func describeContainer(containers []types.Container, containerId string) (*types.Container, error) {
	if containerId == "" {
		var containerNames []string
		for _, container := range containers {
			containerNames = append(containerNames, *container.Name)
		}
		selectedContainerName, err := pterm.DefaultInteractiveSelect.WithOptions(containerNames).Show("Container")
		if err != nil {
			return nil, err
		}
		containerId = selectedContainerName
	}
	for _, container := range containers {
		if *container.Name == containerId {
			return &container, nil
		}
	}
	return nil, fmt.Errorf("no container '%v' found", containerId)
}

func describeContainerDefinition(ctx context.Context, client *ecs.Client, taskDefinitionArn string, containerId string) (*types.ContainerDefinition, error) {
	describedTaskDefinition, err := client.DescribeTaskDefinition(ctx, &ecs.DescribeTaskDefinitionInput{
		TaskDefinition: &taskDefinitionArn,
	})
	if err != nil {
		return nil, err
	}
	containerDefinitions := describedTaskDefinition.TaskDefinition.ContainerDefinitions
	selectedContainerName, err := selectContainerName(containerDefinitions, containerId)
	if err != nil {
		return nil, err
	}
	for _, containerDefinition := range containerDefinitions {
		if *containerDefinition.Name == selectedContainerName {
			return &containerDefinition, nil
		}
	}
	return nil, fmt.Errorf("no container '%v' found", selectedContainerName)
}

func selectContainerName(containerDefinitions []types.ContainerDefinition, containerId string) (string, error) {
	if containerId == "" {
		var containerNames []string
		for _, containerDefinition := range containerDefinitions {
			containerNames = append(containerNames, *containerDefinition.Name)
		}
		return pterm.DefaultInteractiveSelect.WithOptions(containerNames).Show("Container")
	}
	return containerId, nil
}
