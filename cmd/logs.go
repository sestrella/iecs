package cmd

import (
	"context"
	"fmt"
	"log"
	"time"

	"github.com/aws/aws-sdk-go-v2/config"
	logs "github.com/aws/aws-sdk-go-v2/service/cloudwatchlogs"
	logsTypes "github.com/aws/aws-sdk-go-v2/service/cloudwatchlogs/types"
	"github.com/aws/aws-sdk-go-v2/service/ecs"
	"github.com/aws/aws-sdk-go-v2/service/ecs/types"
	"github.com/sestrella/iecs/client"
	"github.com/sestrella/iecs/selector"
	"github.com/spf13/cobra"
)

type ServiceSelection struct {
	Cluster *types.Cluster
	Service *types.Service
}

type SelectedContainerDefinition struct {
	Cluster             *types.Cluster
	Service             *types.Service
	ContainerDefinition *types.ContainerDefinition
}

var logsCmd = &cobra.Command{
	Use:   "logs",
	Short: "View the logs of a container",
	Example: `
  aws-vault exec <profile> -- iecs logs [flags] (recommended)
  env AWS_PROFILE=<profile> iecs logs [flags]
  `,
	RunE: func(cmd *cobra.Command, args []string) error {
		containerMode, err := cmd.Flags().GetBool("container")
		if err != nil {
			return err
		}

		cfg, err := config.LoadDefaultConfig(context.TODO())
		if err != nil {
			return err
		}
		client := client.NewClient(cfg)
		ecsClient := ecs.NewFromConfig(cfg)
		logsClient := logs.NewFromConfig(cfg)
		err = runLogs(
			context.TODO(),
			client,
			selector.NewSelectors(ecsClient),
			ecsClient,
			logsClient,
			containerMode,
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
	client client.Client,
	selectors selector.Selectors,
	ecsClient *ecs.Client,
	logsClient *logs.Client,
	containerMode bool,
) error {
	log.Printf("Container mode: %v", containerMode)
	if containerMode {
		selection, err := containerDefinitionSelector(ctx, selectors)
		if err != nil {
			return err
		}

		if selection.ContainerDefinition.LogConfiguration == nil {
			return fmt.Errorf(
				"no log configuration found for container: %s",
				*selection.ContainerDefinition.Name,
			)
		}

		logOptions := selection.ContainerDefinition.LogConfiguration.Options
		if len(logOptions) == 0 {
			return fmt.Errorf(
				"missing log options for container: %s",
				*selection.ContainerDefinition.Name,
			)
		}

		awslogsGroup, ok := logOptions["awslogs-group"]
		if !ok {
			return fmt.Errorf(
				"missing awslogs-group option for container: %s",
				*selection.ContainerDefinition.Name,
			)
		}

		streamPrefix, ok := logOptions["awslogs-stream-prefix"]
		if !ok {
			return fmt.Errorf(
				"missing awslogs-stream-prefix option for container: %s",
				*selection.ContainerDefinition.Name,
			)
		}

		// Use our logs client to start the live tail
		err = client.StartLiveTail(
			ctx,
			awslogsGroup,
			streamPrefix,
			func(timestamp time.Time, message string) {
				fmt.Printf("%v %s\n", timestamp, message)
			},
		)
		if err != nil {
			return err
		}

		return nil
	}

	cluster, err := selectors.Cluster(ctx)
	if err != nil {
		return err
	}

	service, err := selectors.Service(ctx, cluster)
	if err != nil {
		return err
	}

	describedTaskDefinition, err := ecsClient.DescribeTaskDefinition(
		ctx,
		&ecs.DescribeTaskDefinitionInput{
			TaskDefinition: service.TaskDefinition,
		},
	)
	if err != nil {
		return err
	}

	var logGroupArns []string
	for _, containerDefinition := range describedTaskDefinition.TaskDefinition.ContainerDefinitions {
		logGroupName := containerDefinition.LogConfiguration.Options["awslogs-group"]
		describedLogGroups, err := logsClient.DescribeLogGroups(ctx, &logs.DescribeLogGroupsInput{
			LogGroupNamePattern: &logGroupName,
		})
		if err != nil {
			return err
		}
		logGroupArns = append(logGroupArns, *describedLogGroups.LogGroups[0].LogGroupArn)
	}

	startedLiveTail, err := logsClient.StartLiveTail(ctx, &logs.StartLiveTailInput{
		LogGroupIdentifiers: logGroupArns,
	})
	if err != nil {
		return err
	}

	stream := startedLiveTail.GetStream()
	defer func() {
		err := stream.Close()
		if err != nil {
			log.Fatal(err)
		}
	}()

	eventsChan := stream.Events()
	for {
		event := <-eventsChan
		switch e := event.(type) {
		case *logsTypes.StartLiveTailResponseStreamMemberSessionUpdate:
			for _, logEvent := range e.Value.SessionResults {
				eventDate := time.UnixMilli(*logEvent.Timestamp)
				fmt.Printf("%s %s %s\n", eventDate, *logEvent.Message, *logEvent.LogStreamName)
			}
		}
	}
}

func containerDefinitionSelector(
	ctx context.Context,
	selectors selector.Selectors,
) (*SelectedContainerDefinition, error) {
	cluster, err := selectors.Cluster(ctx)
	if err != nil {
		return nil, err
	}

	service, err := selectors.Service(ctx, cluster)
	if err != nil {
		return nil, err
	}

	containerDefinition, err := selectors.ContainerDefinition(ctx, service)
	if err != nil {
		return nil, err
	}

	return &SelectedContainerDefinition{
		Cluster:             cluster,
		Service:             service,
		ContainerDefinition: containerDefinition,
	}, nil
}

func init() {
	logsCmd.Flags().Bool("container", false, "")
	rootCmd.AddCommand(logsCmd)
}
