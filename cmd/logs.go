package cmd

import (
	"context"
	"fmt"
	"log"
	"strings"
	"sync"
	"time"

	"github.com/aws/aws-sdk-go-v2/config"
	logsTypes "github.com/aws/aws-sdk-go-v2/service/cloudwatchlogs/types"
	"github.com/aws/aws-sdk-go-v2/service/ecs/types"
	"github.com/fatih/color"
	"github.com/sestrella/iecs/client"
	"github.com/sestrella/iecs/selector"
	"github.com/spf13/cobra"
)

var printers = []Printer{
	color.Blue,
	color.Cyan,
	color.Magenta,
	color.Red,
}

type Printer = func(string, ...any)

type LogsSelection struct {
	cluster    *types.Cluster
	service    *types.Service
	tasks      []types.Task
	containers []types.ContainerDefinition
}

var logsCmd = &cobra.Command{
	Use:   "logs",
	Short: "View the logs of a container",
	Example: `
  aws-vault exec <profile> -- iecs logs [flags] (recommended)
  env AWS_PROFILE=<profile> iecs logs [flags]
  `,
	RunE: func(cmd *cobra.Command, args []string) error {
		noColors, err := cmd.Flags().GetBool("no-colors")
		if err != nil {
			return err
		}

		cfg, err := config.LoadDefaultConfig(context.TODO())
		if err != nil {
			return err
		}

		client := client.NewClient(cfg)

		clusterPattern, err := cmd.Flags().GetString("cluster")
		if err != nil {
			return err
		}

		servicePattern, err := cmd.Flags().GetString("service")
		if err != nil {
			return err
		}

		err = runLogs(
			context.TODO(),
			noColors,
			client,
			selector.NewSelectors(client, cmd.Flag("theme").Value.String()),
			clusterPattern,
			servicePattern,
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
	noColors bool,
	clients client.Client,
	selectors selector.Selectors,
	clusterPattern string,
	servicePattern string,
) error {
	type LogOptions struct {
		containerName string
		group         string
		streamPrefix  string
		printer       Printer
	}

	selection, err := logsSelector(ctx, selectors, clusterPattern, servicePattern)
	if err != nil {
		return err
	}

	var allLogOptions []LogOptions
	for index, container := range selection.containers {
		if container.LogConfiguration == nil {
			return fmt.Errorf("no log configuration found for container %s", *container.Name)
		}
		options := container.LogConfiguration.Options
		if options == nil {
			return fmt.Errorf("no log options found for container %s", *container.Name)
		}
		allLogOptions = append(allLogOptions, LogOptions{
			containerName: *container.Name,
			group:         options["awslogs-group"],
			streamPrefix:  options["awslogs-stream-prefix"],
			printer:       printerByIndex(noColors, index),
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

			go func(taskId string, logOptions LogOptions) {
				defer wg.Done()

				err := clients.StartLiveTail(
					ctx,
					logOptions.group,
					streamName,
					client.LiveTailHandlers{
						Start: func() {
							log.Printf(
								"Starting live tail for container '%s' running at task '%s'\n",
								logOptions.containerName,
								taskId,
							)
						},
						Update: func(event logsTypes.LiveTailSessionLogEvent) {
							timestamp := time.UnixMilli(*event.Timestamp)
							if len(selection.tasks) > 1 {
								logOptions.printer(
									"%s | %s | %s | %s\n",
									taskId,
									logOptions.containerName,
									timestamp,
									*event.Message,
								)
							} else if len(selection.containers) > 1 {
								logOptions.printer(
									"%s | %s | %s\n",
									logOptions.containerName,
									timestamp,
									*event.Message,
								)
							} else {
								logOptions.printer(
									"%s | %s\n",
									timestamp,
									*event.Message,
								)
							}
						},
					},
				)
				if err != nil {
					logOptions.printer("Error live tailing logs: %v", err)
				}
			}(taskId, logOptions)
		}
	}
	wg.Wait()

	return nil
}

func logsSelector(
	ctx context.Context,
	selectors selector.Selectors,
	clusterPattern string,
	servicePattern string,
) (*LogsSelection, error) {
	cluster, err := selectors.Cluster(ctx, clusterPattern)
	if err != nil {
		return nil, err
	}

	service, err := selectors.Service(ctx, cluster, servicePattern)
	if err != nil {
		return nil, err
	}

	tasks, err := selectors.Tasks(ctx, service)
	if err != nil {
		return nil, err
	}

	containers, err := selectors.ContainerDefinitions(ctx, *service.TaskDefinition)
	if err != nil {
		return nil, err
	}

	return &LogsSelection{
		cluster:    cluster,
		service:    service,
		tasks:      tasks,
		containers: containers,
	}, nil
}

func printerByIndex(noColors bool, index int) Printer {
	if noColors {
		return func(format string, a ...any) {
			fmt.Printf(format, a...)
		}
	}

	return printers[index%len(printers)]
}

func init() {
	rootCmd.AddCommand(logsCmd)

	logsCmd.Flags().BoolP("no-colors", "", false, "Disable log coloring")
}
