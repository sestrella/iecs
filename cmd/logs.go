/*
Copyright © 2024 NAME HERE <EMAIL ADDRESS>
*/
package cmd

import (
	"context"
	"fmt"
	"log"
	"sync"

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

		listClusters, _ := ecsClient.ListClusters(context.TODO(), &ecs.ListClustersInput{})
		clusterArn, _ := pterm.DefaultInteractiveSelect.WithOptions(listClusters.ClusterArns).Show("Cluster")
		listTasks, _ := ecsClient.ListTasks(context.TODO(), &ecs.ListTasksInput{
			Cluster: &clusterArn,
		})
		taskArn, _ := pterm.DefaultInteractiveSelect.WithOptions(listTasks.TaskArns).Show("Task")
		tasks, _ := ecsClient.DescribeTasks(context.TODO(), &ecs.DescribeTasksInput{
			Cluster: &clusterArn,
			Tasks:   []string{taskArn},
		})
		task := tasks.Tasks[0]
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
						log.Println(*logEvent.Message)
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

	rootCmd.Flags().String("cluster", "", "TODO")

	// Here you will define your flags and configuration settings.

	// Cobra supports Persistent Flags which will work for this command
	// and all subcommands, e.g.:
	// logsCmd.PersistentFlags().String("foo", "", "A help for foo")

	// Cobra supports local flags which will only run when this command
	// is called directly, e.g.:
	// logsCmd.Flags().BoolP("toggle", "t", false, "Help message for toggle")
}
