package cmd

import (
	"context"
	"fmt"
	"strings"
	"sync"
	"time"

	"github.com/aws/aws-sdk-go-v2/config"
	"github.com/aws/aws-sdk-go-v2/service/cloudwatchlogs"
	logsTypes "github.com/aws/aws-sdk-go-v2/service/cloudwatchlogs/types"
	"github.com/aws/aws-sdk-go-v2/service/ecs"
	"github.com/aws/aws-sdk-go-v2/service/ecs/types"
	"github.com/fatih/color"
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
	log           func(format string, args ...any)
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
	client client.Client,
	selectors selector.Selectors,
) error {
	selection, err := containerDefinitionSelector(ctx, ecsClient, selectors)
	if err != nil {
		return err
	}

	var allLogOptions []LogOptions
	for index, container := range selection.containers {
		options := container.LogConfiguration.Options
		// TODO: check if options exist
		allLogOptions = append(allLogOptions, LogOptions{
			containerName: *container.Name,
			group:         options["awslogs-group"],
			streamPrefix:  options["awslogs-stream-prefix"],
			log:           logByIndex(index),
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
				logGroups, err := logsClient.DescribeLogGroups(
					ctx,
					&cloudwatchlogs.DescribeLogGroupsInput{
						LogGroupNamePrefix: &logOptions.group,
					},
				)
				// TODO check log groups size

				startLiveTail, err := logsClient.StartLiveTail(
					ctx,
					&cloudwatchlogs.StartLiveTailInput{
						LogGroupIdentifiers: []string{*logGroups.LogGroups[0].LogGroupArn},
						LogStreamNames:      []string{streamName},
					},
				)
				if err != nil {
					panic(err)
				}

				stream := startLiveTail.GetStream()
				defer stream.Close()

				events := stream.Events()
				for {
					event := <-events
					switch e := event.(type) {
					case *logsTypes.StartLiveTailResponseStreamMemberSessionStart:
						fmt.Println("Received SessionStart event")
					case *logsTypes.StartLiveTailResponseStreamMemberSessionUpdate:
						for _, result := range e.Value.SessionResults {
							timestamp := time.UnixMilli(*result.Timestamp)
							logOptions.log("%s | %s | %s | %s\n", taskId, logOptions.containerName, timestamp, *result.Message)
						}
					default:
						if err := stream.Err(); err != nil {
							fmt.Printf("Error occured during streaming: %v", err)
						} else if event == nil {
							fmt.Println("Stream is Closed")
							return
						} else {
							fmt.Printf("Unknown event type: %T", e)
						}
					}
				}
			}()
		}
	}
	wg.Wait()

	return nil
}

func logByIndex(index int) func(string, ...any) {
	switch index % 3 {
	case 0:
		return color.Cyan
	case 1:
		return color.Blue
	default:
		return color.Magenta
	}
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
