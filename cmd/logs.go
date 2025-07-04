package cmd

import (
	"context"
	"fmt"
	"log"
	"strings"
	"sync"
	"time"

	"github.com/aws/aws-sdk-go-v2/config"
	"github.com/aws/aws-sdk-go-v2/service/cloudwatchlogs"
	logsTypes "github.com/aws/aws-sdk-go-v2/service/cloudwatchlogs/types"
	"github.com/aws/aws-sdk-go-v2/service/ecs"
	"github.com/aws/aws-sdk-go-v2/service/ecs/types"
	"github.com/sestrella/iecs/client"
	"github.com/sestrella/iecs/selector"
	"github.com/spf13/cobra"
)

type LogsSelection struct {
	cluster    *types.Cluster
	service    *types.Service
	tasks      []types.Task
	containers []types.ContainerDefinition
}

type LogOptions struct {
	containerName string
	group         string
	streamPrefix  string
}

var logsCmd = &cobra.Command{
	Use:   "logs",
	Short: "View the logs of a container",
	Example: `
  aws-vault exec <profile> -- iecs logs [flags] (recommended)
  env AWS_PROFILE=<profile> iecs logs [flags]
  `,
	RunE: func(cmd *cobra.Command, args []string) error {
		cfg, err := config.LoadDefaultConfig(context.TODO())
		if err != nil {
			return err
		}

		ecsClient := ecs.NewFromConfig(cfg)
		logsClient := cloudwatchlogs.NewFromConfig(cfg)
		client := client.NewClient(cfg)
		err = runLogs(
			context.TODO(),
			ecsClient,
			logsClient,
			client,
			selector.NewSelectors(ecsClient),
		)
		if err != nil {
			return err
		}

		return nil
	},
	Aliases: []string{"tail"},
}

func runLogs(
	ctx context.Context,
	ecsClient *ecs.Client,
	logsClient *cloudwatchlogs.Client,
	foo client.Client,
	selectors selector.Selectors,
) error {
	selection, err := containerDefinitionSelector(ctx, ecsClient, selectors)
	if err != nil {
		return err
	}

	var allLogOptions []LogOptions
	for _, container := range selection.containers {
		options := container.LogConfiguration.Options
		// TODO: check if options exist
		allLogOptions = append(allLogOptions, LogOptions{
			containerName: *container.Name,
			group:         options["awslogs-group"],
			streamPrefix:  options["awslogs-stream-prefix"],
		})
	}

	var wg sync.WaitGroup
	for _, task := range selection.tasks {
		taskArnSlices := strings.Split(*task.TaskArn, "/")
		taskId := taskArnSlices[len(taskArnSlices)-1]

		for _, logOptions := range allLogOptions {
			streamName := fmt.Sprintf(
				"%s/%s/%s",
				logOptions.streamPrefix,
				logOptions.containerName,
				taskId,
			)
			wg.Add(1)

			go func() {
				defer wg.Done()

				foo.StartLiveTail(ctx, logOptions.group, streamName, client.LiveTailHandlers{
					Start: func() {
						log.Printf(
							"Starting live trail for container '%s' running at task '%s'\n",
							logOptions.containerName,
							taskId,
						)
					},
					Update: func(event logsTypes.LiveTailSessionLogEvent) {
						timestamp := time.UnixMilli(*event.Timestamp)
						log.Printf(
							"%s | %s | %s | %s\n",
							taskId,
							logOptions.containerName,
							timestamp,
							*event.Message,
						)
					},
				})
			}()
		}
	}
	wg.Wait()

	return nil
}

func containerDefinitionSelector(
	ctx context.Context,
	ecsClient *ecs.Client,
	selectors selector.Selectors,
) (*LogsSelection, error) {
	cluster, err := selectors.Cluster(ctx)
	if err != nil {
		return nil, err
	}

	service, err := selectors.Service(ctx, cluster)
	if err != nil {
		return nil, err
	}

	listTasks, err := ecsClient.ListTasks(ctx, &ecs.ListTasksInput{
		Cluster:     cluster.ClusterArn,
		ServiceName: service.ServiceName,
	})
	if err != nil {
		return nil, err
	}

	describeTasks, err := ecsClient.DescribeTasks(ctx, &ecs.DescribeTasksInput{
		Cluster: cluster.ClusterArn,
		Tasks:   listTasks.TaskArns,
	})

	containers, err := selectors.ContainerDefinitions(ctx, *service.TaskDefinition)
	if err != nil {
		return nil, err
	}

	return &LogsSelection{
		cluster:    cluster,
		service:    service,
		tasks:      describeTasks.Tasks,
		containers: containers,
	}, nil
}

func init() {
	rootCmd.AddCommand(logsCmd)
}
