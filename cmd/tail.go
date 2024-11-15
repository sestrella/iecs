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

const TAIL_CLUSTER_FLAG = "cluster"
const TAIL_SERVICE_FLAG = "service"
const TAIL_TASK_FLAG = "task"
const TAIL_CONTAINER_FLAG = "container"

var tailCmd = &cobra.Command{
	Use:   "tail",
	Short: "View the logs of a container",
	Example: `
  aws-vault exec <profile> -- iecs tail (recommended)
  env AWS_PROFILE=<profile> iecs tail
  `,
	Run: func(cmd *cobra.Command, args []string) {
		clusterId, err := cmd.Flags().GetString(TAIL_CLUSTER_FLAG)
		if err != nil {
			log.Fatal(err)
		}
		serviceId, err := cmd.Flags().GetString(TAIL_SERVICE_FLAG)
		if err != nil {
			log.Fatal(err)
		}
		taskId, err := cmd.Flags().GetString(TAIL_TASK_FLAG)
		if err != nil {
			log.Fatal(err)
		}
		containerId, err := cmd.Flags().GetString(TAIL_CONTAINER_FLAG)
		if err != nil {
			log.Fatal(err)
		}
		cfg, err := config.LoadDefaultConfig(context.TODO())
		if err != nil {
			log.Fatal(err)
		}
		ecsClient := ecs.NewFromConfig(cfg)
		cwlogsClient := cloudwatchlogs.NewFromConfig(cfg)
		err = runTail(context.TODO(), ecsClient, cwlogsClient, clusterId, serviceId, taskId, containerId)
		if err != nil {
			log.Fatal(err)
		}
	},
	Aliases: []string{"logs"},
}

func runTail(ctx context.Context, ecsClient *ecs.Client, cwlogsClient *cloudwatchlogs.Client, clusterId string, serviceId string, taskId string, containerId string) error {
	cluster, err := describeCluster(ctx, ecsClient, clusterId)
	if err != nil {
		return err
	}
	service, err := describeService(ctx, ecsClient, *cluster.ClusterArn, serviceId)
	if err != nil {
		return err
	}
	task, err := describeTask(context.TODO(), ecsClient, *cluster.ClusterArn, *service.ServiceName, taskId)
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
	rootCmd.AddCommand(tailCmd)

	tailCmd.Flags().StringP(TAIL_CLUSTER_FLAG, "c", "", "cluster id or ARN")
	tailCmd.Flags().StringP(TAIL_SERVICE_FLAG, "s", "", "service id or ARN")
	tailCmd.Flags().StringP(TAIL_TASK_FLAG, "t", "", "task id or ARN")
	tailCmd.Flags().StringP(TAIL_CONTAINER_FLAG, "n", "", "container id")
}
