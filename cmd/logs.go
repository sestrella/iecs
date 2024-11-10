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
	"github.com/aws/aws-sdk-go-v2/service/ecs/types"
	"github.com/aws/aws-sdk-go-v2/service/sts"
	"github.com/pterm/pterm"
	"github.com/spf13/cobra"
)

// logsCmd represents the logs command
var logsCmd = &cobra.Command{
	Use:   "logs",
	Short: "A brief description of your command",
	Run: func(cmd *cobra.Command, args []string) {
		cfg, _ := config.LoadDefaultConfig(context.TODO())

		ecsClient := ecs.NewFromConfig(cfg)
		cwlogsClient := cloudwatchlogs.NewFromConfig(cfg)
		stsClient := sts.NewFromConfig(cfg)

		cluster, err := describeCluster(context.TODO(), ecsClient, nil)
		if err != nil {
			log.Fatal(err)
		}
		task, err := describeTask(context.TODO(), ecsClient, cluster.ClusterArn, nil)
		if err != nil {
			log.Fatal(err)
		}

		taskDefinition, _ := ecsClient.DescribeTaskDefinition(context.TODO(), &ecs.DescribeTaskDefinitionInput{
			TaskDefinition: task.TaskDefinitionArn,
		})

		var containerNames []string
		for _, container := range taskDefinition.TaskDefinition.ContainerDefinitions {
			containerNames = append(containerNames, *container.Name)
		}
		containerName, _ := pterm.DefaultInteractiveSelect.WithOptions(containerNames).Show("Container")
		var selectedContainer *types.ContainerDefinition
		for _, container := range taskDefinition.TaskDefinition.ContainerDefinitions {
			if *container.Name == containerName {
				selectedContainer = &container
				break
			}
		}
		logOptions := selectedContainer.LogConfiguration.Options
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

func describeCluster(ctx context.Context, client *ecs.Client, clusterId *string) (*types.Cluster, error) {
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

func selectClusterId(ctx context.Context, client *ecs.Client, clusterId *string) (*string, error) {
	if clusterId == nil {
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
	return clusterId, nil
}

func describeTask(ctx context.Context, client *ecs.Client, clusterId *string, taskId *string) (*types.Task, error) {
	selectedTaskId, err := selectTaskId(ctx, client, clusterId, taskId)
	if err != nil {
		return nil, err
	}
	describedTasks, err := client.DescribeTasks(ctx, &ecs.DescribeTasksInput{
		Cluster: clusterId,
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

func selectTaskId(ctx context.Context, client *ecs.Client, clusterId *string, taskId *string) (*string, error) {
	if taskId == nil {
		listTasks, _ := client.ListTasks(ctx, &ecs.ListTasksInput{
			Cluster: clusterId,
		})
		taskArn, err := pterm.DefaultInteractiveSelect.WithOptions(listTasks.TaskArns).Show("Task")
		if err != nil {
			return nil, err
		}
		return &taskArn, nil
	}
	return taskId, nil
}

func init() {
	rootCmd.AddCommand(logsCmd)

	rootCmd.Flags().String("cluster", "", "TODO")
	rootCmd.Flags().String("task", "", "TODO")
	rootCmd.Flags().String("container", "", "TODO")

	// Here you will define your flags and configuration settings.

	// Cobra supports Persistent Flags which will work for this command
	// and all subcommands, e.g.:
	// logsCmd.PersistentFlags().String("foo", "", "A help for foo")

	// Cobra supports local flags which will only run when this command
	// is called directly, e.g.:
	// logsCmd.Flags().BoolP("toggle", "t", false, "Help message for toggle")
}
