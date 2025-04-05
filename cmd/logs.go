package cmd

import (
	"context"
	"fmt"
	"log"
	"time"

	"github.com/aws/aws-sdk-go-v2/config"
	cwlogs "github.com/aws/aws-sdk-go-v2/service/cloudwatchlogs"
	cwlogsTypes "github.com/aws/aws-sdk-go-v2/service/cloudwatchlogs/types"
	"github.com/aws/aws-sdk-go-v2/service/ecs"
	"github.com/sestrella/iecs/selector"
	"github.com/spf13/cobra"
)

const (
	logsClusterFlag   = "cluster"
	logsServiceFlag   = "service"
	logsContainerFlag = "container"
)

var logsCmd = &cobra.Command{
	Use:   "logs",
	Short: "View the logs of a container",
	Example: `
  aws-vault exec <profile> -- iecs logs [flags] (recommended)
  env AWS_PROFILE=<profile> iecs logs [flags]
  `,
	Run: func(cmd *cobra.Command, args []string) {
		clusterId, err := cmd.Flags().GetString(logsClusterFlag)
		if err != nil {
			panic(err)
		}
		serviceId, err := cmd.Flags().GetString(logsServiceFlag)
		if err != nil {
			panic(err)
		}
		containerId, err := cmd.Flags().GetString(logsContainerFlag)
		if err != nil {
			panic(err)
		}
		cfg, err := config.LoadDefaultConfig(context.TODO())
		if err != nil {
			panic(err)
		}
		ecsClient := ecs.NewFromConfig(cfg)
		cwlogsClient := cwlogs.NewFromConfig(cfg)
		err = runLogs(context.TODO(), ecsClient, cwlogsClient, clusterId, serviceId, containerId)
		if err != nil {
			panic(err)
		}
	},
	Aliases: []string{"tail"},
}

func runLogs(
	ctx context.Context,
	ecsClient *ecs.Client,
	cwlogsClient *cwlogs.Client,
	clusterId string,
	serviceId string,
	containerId string,
) error {
	defaultSelector := &selector.DefaultSelector{}
	cluster, err := selector.SelectCluster(ctx, ecsClient, defaultSelector, clusterId)
	if err != nil {
		return err
	}
	service, err := selector.SelectService(
		ctx,
		ecsClient,
		defaultSelector,
		*cluster.ClusterArn,
		serviceId,
	)
	if err != nil {
		return err
	}
	container, err := selector.SelectContainerDefinition(
		context.TODO(),
		ecsClient,
		*service.TaskDefinition,
		containerId,
	)
	if err != nil {
		return err
	}
	logOptions := container.LogConfiguration.Options
	awslogsGroup := logOptions["awslogs-group"]
	describeLogGroups, err := cwlogsClient.DescribeLogGroups(
		context.TODO(),
		&cwlogs.DescribeLogGroupsInput{
			LogGroupNamePrefix: &awslogsGroup,
		},
	)
	if err != nil {
		return err
	}
	startLiveTail, err := cwlogsClient.StartLiveTail(context.TODO(), &cwlogs.StartLiveTailInput{
		LogGroupIdentifiers:   []string{*describeLogGroups.LogGroups[0].LogGroupArn},
		LogStreamNamePrefixes: []string{logOptions["awslogs-stream-prefix"]},
	})
	if err != nil {
		return err
	}
	stream := startLiveTail.GetStream()
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
			panic(fmt.Sprintf("Unknown event type: %s", e))
		}
	}
}

func init() {
	rootCmd.AddCommand(logsCmd)

	logsCmd.Flags().StringP(logsClusterFlag, "l", "", "cluster id or ARN")
	logsCmd.Flags().StringP(logsServiceFlag, "s", "", "service id or ARN")
	logsCmd.Flags().StringP(logsContainerFlag, "n", "", "container name")
}
