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
	"github.com/sestrella/iecs/selector"
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
		clusterId, err := cmd.Flags().GetString(CLUSTER_FLAG)
		if err != nil {
			log.Fatal(err)
		}
		serviceId, err := cmd.Flags().GetString(SERVICE_FLAG)
		if err != nil {
			log.Fatal(err)
		}
		containerId, err := cmd.Flags().GetString(CONTAINER_FLAG)
		if err != nil {
			log.Fatal(err)
		}
		cfg, err := config.LoadDefaultConfig(context.TODO())
		if err != nil {
			log.Fatal(err)
		}
		ecsClient := ecs.NewFromConfig(cfg)
		cwlogsClient := cloudwatchlogs.NewFromConfig(cfg)
		err = runLogs(context.TODO(), ecsClient, cwlogsClient, clusterId, serviceId, containerId)
		if err != nil {
			log.Fatal(err)
		}
	},
	Aliases: []string{"tail"},
}

func runLogs(ctx context.Context, ecsClient *ecs.Client, cwlogsClient *cloudwatchlogs.Client, clusterId string, serviceId string, containerId string) error {
	cluster, err := selector.SelectCluster(ctx, ecsClient, clusterId)
	if err != nil {
		return err
	}
	service, err := selector.SelectService(ctx, ecsClient, *cluster.ClusterArn, serviceId)
	if err != nil {
		return err
	}
	container, err := selector.SelectContainerDefinition(context.TODO(), ecsClient, *service.TaskDefinition, containerId)
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

func init() {
	rootCmd.AddCommand(logsCmd)
}
