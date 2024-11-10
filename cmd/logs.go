/*
Copyright © 2024 NAME HERE <EMAIL ADDRESS>
*/
package cmd

import (
	"context"
	"fmt"
	"log"
	"sync"
	"time"

	"github.com/aws/aws-sdk-go-v2/config"
	"github.com/aws/aws-sdk-go-v2/service/cloudwatchlogs"
	cwlogsTypes "github.com/aws/aws-sdk-go-v2/service/cloudwatchlogs/types"
	"github.com/aws/aws-sdk-go-v2/service/ecs"
	ecsTypes "github.com/aws/aws-sdk-go-v2/service/ecs/types"
	"github.com/aws/aws-sdk-go-v2/service/sts"
	"github.com/pterm/pterm"
	"github.com/spf13/cobra"
)

// logsCmd represents the logs command
var logsCmd = &cobra.Command{
	Use:   "logs",
	Short: "A brief description of your command",
	Run: func(cmd *cobra.Command, args []string) {
		clusterId, err := cmd.Flags().GetString("cluster")
		if err != nil {
			log.Fatal(err)
		}
		taskId, err := cmd.Flags().GetString("task")
		if err != nil {
			log.Fatal(err)
		}

		cfg, err := config.LoadDefaultConfig(context.TODO())
		if err != nil {
			log.Fatal(err)
		}

		ecsClient := ecs.NewFromConfig(cfg)
		cwlogsClient := cloudwatchlogs.NewFromConfig(cfg)
		stsClient := sts.NewFromConfig(cfg)

		cluster, err := describeCluster(context.TODO(), ecsClient, clusterId)
		if err != nil {
			log.Fatal(err)
		}
		task, err := describeTask(context.TODO(), ecsClient, *cluster.ClusterArn, taskId)
		if err != nil {
			log.Fatal(err)
		}
		container, err := describeContainerDefinition(ecsClient, *task.TaskDefinitionArn)
		if err != nil {
			log.Fatal(err)
		}
		logOptions := container.LogConfiguration.Options
		getCallerIdentity, _ := stsClient.GetCallerIdentity(context.TODO(), &sts.GetCallerIdentityInput{})

		// TODO: get log_group ARN from SDK
		startLiveTail, _ := cwlogsClient.StartLiveTail(context.TODO(), &cloudwatchlogs.StartLiveTailInput{
			LogGroupIdentifiers:   []string{fmt.Sprintf("arn:aws:logs:%s:%s:log-group:%s", logOptions["awslogs-region"], *getCallerIdentity.Account, logOptions["awslogs-group"])},
			LogStreamNamePrefixes: []string{logOptions["awslogs-stream-prefix"]},
		})
		stream := startLiveTail.GetStream()
		var wg sync.WaitGroup
		wg.Add(1)
		go func() {
			defer wg.Done()
			eventsChannel := stream.Events()
			for {
				event := <-eventsChannel
				switch e := event.(type) {
				case *cwlogsTypes.StartLiveTailResponseStreamMemberSessionStart:
					log.Println("Received SessionStart event")
				case *cwlogsTypes.StartLiveTailResponseStreamMemberSessionUpdate:
					for _, logEvent := range e.Value.SessionResults {
						date := time.UnixMilli(*logEvent.Timestamp)
						fmt.Printf("%v %s\n", date, *logEvent.Message)
					}
				default:
					fmt.Println("TODO")
					return
				}
			}
		}()
		wg.Wait()
	},
}

func describeCluster(ctx context.Context, client *ecs.Client, clusterId string) (*ecsTypes.Cluster, error) {
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
	clustersCount := len(describedClusters.Clusters)
	if clustersCount == 1 {
		return &describedClusters.Clusters[0], nil
	}
	return nil, fmt.Errorf("expect 1 cluster, got %v", clustersCount)
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

func describeTask(ctx context.Context, client *ecs.Client, clusterId string, taskId string) (*ecsTypes.Task, error) {
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
	tasksCount := len(describedTasks.Tasks)
	if tasksCount == 1 {
		return &describedTasks.Tasks[0], nil
	}
	return nil, fmt.Errorf("expect 1 task, got %v", tasksCount)
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

func describeContainerDefinition(ecsClient *ecs.Client, taskDefinitionArn string) (*ecsTypes.ContainerDefinition, error) {
	taskDefinition, err := ecsClient.DescribeTaskDefinition(context.TODO(), &ecs.DescribeTaskDefinitionInput{
		TaskDefinition: &taskDefinitionArn,
	})
	if err != nil {
		return nil, err
	}
	var containerNames []string
	for _, container := range taskDefinition.TaskDefinition.ContainerDefinitions {
		containerNames = append(containerNames, *container.Name)
	}
	containerName, err := pterm.DefaultInteractiveSelect.WithOptions(containerNames).Show("Container")
	if err != nil {
		return nil, err
	}
	for _, container := range taskDefinition.TaskDefinition.ContainerDefinitions {
		if *container.Name == containerName {
			return &container, nil
		}
	}
	return nil, fmt.Errorf("no container found")
}

func init() {
	rootCmd.AddCommand(logsCmd)

	logsCmd.Flags().String("cluster", "", "cluster id or ARN")
	logsCmd.Flags().String("task", "", "task id or ARN")
	logsCmd.Flags().String("container", "", "container id")
}
