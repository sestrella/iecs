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
	"github.com/spf13/cobra"
)

// logsCmd represents the logs command
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

		cluster, err := describeCluster(context.TODO(), ecsClient, clusterId)
		if err != nil {
			log.Fatal(err)
		}
		task, err := describeTask(context.TODO(), ecsClient, *cluster.ClusterArn, taskId)
		if err != nil {
			log.Fatal(err)
		}
		container, err := describeContainerDefinition(context.TODO(), ecsClient, *task.TaskDefinitionArn, containerId)
		if err != nil {
			log.Fatal(err)
		}
		logOptions := container.LogConfiguration.Options
		awslogsGroup := logOptions["awslogs-group"]
		describeLogGroups, err := cwlogsClient.DescribeLogGroups(context.TODO(), &cloudwatchlogs.DescribeLogGroupsInput{
			LogGroupNamePrefix: &awslogsGroup,
		})
		if err != nil {
			log.Fatal(err)
		}
		startLiveTail, err := cwlogsClient.StartLiveTail(context.TODO(), &cloudwatchlogs.StartLiveTailInput{
			LogGroupIdentifiers:   []string{*describeLogGroups.LogGroups[0].LogGroupArn},
			LogStreamNamePrefixes: []string{logOptions["awslogs-stream-prefix"]},
		})
		if err != nil {
			log.Fatal(err)
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
	},
}

func init() {
	rootCmd.AddCommand(logsCmd)

	logsCmd.Flags().StringP("cluster", "c", "", "cluster id or ARN")
	logsCmd.Flags().StringP("task", "t", "", "task id or ARN")
	logsCmd.Flags().StringP("container", "n", "", "container id")
}
