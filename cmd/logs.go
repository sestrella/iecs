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
	"github.com/pterm/pterm"
	"github.com/spf13/cobra"
)

var logsCmd = &cobra.Command{
	Use:   "logs",
	Short: "View the logs of a container",
	Example: `
  aws-vault exec <profile> -- iecs logs (recommended)
  env AWS_PROFILE=<profile> iecs logs
  `,
	Run: func(cmd *cobra.Command, args []string) {
		clusterId, err := cmd.Flags().GetString("cluster")
		if err != nil {
			log.Fatal(err)
		}
		taskId, err := cmd.Flags().GetString("task")
		if err != nil {
			log.Fatal(err)
		}
		containerId, err := cmd.Flags().GetString("container")
		if err != nil {
			log.Fatal(err)
		}
		cfg, err := config.LoadDefaultConfig(context.TODO())
		if err != nil {
			log.Fatal(err)
		}
		ecsClient := ecs.NewFromConfig(cfg)
		cwlogsClient := cloudwatchlogs.NewFromConfig(cfg)
		err = runLogs(context.TODO(), ecsClient, cwlogsClient, clusterId, taskId, containerId)
		if err != nil {
			log.Fatal(err)
		}
	},
}

func runLogs(ctx context.Context, ecsClient *ecs.Client, cwlogsClient *cloudwatchlogs.Client, clusterId string, taskId string, containerId string) error {
	cluster, err := describeCluster(ctx, ecsClient, clusterId)
	if err != nil {
		return err
	}
	task, err := describeTask(context.TODO(), ecsClient, *cluster.ClusterArn, taskId)
	if err != nil {
		return err
	}
	container, err := describeContainerDefinition(context.TODO(), ecsClient, *task.TaskDefinitionArn, containerId)
	if err != nil {
		return err
	}
	logOptions := container.LogConfiguration.Options
	awslogsGroup := logOptions["awslogs-group"]
	describeLogGroups, err := cwlogsClient.DescribeLogGroups(context.TODO(), &cloudwatchlogs.DescribeLogGroupsInput{
		LogGroupNamePrefix: &awslogsGroup,
	})
	if err != nil {
		return err
	}
	startLiveTail, err := cwlogsClient.StartLiveTail(context.TODO(), &cloudwatchlogs.StartLiveTailInput{
		LogGroupIdentifiers:   []string{*describeLogGroups.LogGroups[0].LogGroupArn},
		LogStreamNamePrefixes: []string{logOptions["awslogs-stream-prefix"]},
	})
	if err != nil {
		return err
	}
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
	return nil
}

func describeContainerDefinition(ctx context.Context, client *ecs.Client, taskDefinitionArn string, containerId string) (*ecsTypes.ContainerDefinition, error) {
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

func init() {
	rootCmd.AddCommand(logsCmd)

	logsCmd.Flags().StringP("cluster", "c", "", "cluster id or ARN")
	logsCmd.Flags().StringP("task", "t", "", "task id or ARN")
	logsCmd.Flags().StringP("container", "n", "", "container id")
}
