//go:build !DEMO

package client

import (
	"context"
	"encoding/json"
	"fmt"
	"log"
	"os/exec"
	"strings"
	"time"

	"github.com/aws/aws-sdk-go-v2/aws"
	logs "github.com/aws/aws-sdk-go-v2/service/cloudwatchlogs"
	logsTypes "github.com/aws/aws-sdk-go-v2/service/cloudwatchlogs/types"
	"github.com/aws/aws-sdk-go-v2/service/ecs"
	ecsTypes "github.com/aws/aws-sdk-go-v2/service/ecs/types"
	"github.com/aws/aws-sdk-go-v2/service/ssm"
)

// awsClient implements the combined Client interface
type awsClient struct {
	region     string
	ecsClient  *ecs.Client
	logsClient *logs.Client
}

// NewClient creates a new combined AWS client
func NewClient(cfg aws.Config) Client {
	ecsClient := ecs.NewFromConfig(cfg)
	logsClient := logs.NewFromConfig(cfg)
	return &awsClient{
		region:     cfg.Region,
		ecsClient:  ecsClient,
		logsClient: logsClient,
	}
}

// ECS operations implementation

func (c *awsClient) ListClusters(ctx context.Context) ([]string, error) {
	listClusters, err := c.ecsClient.ListClusters(ctx, &ecs.ListClustersInput{})
	if err != nil {
		return nil, err
	}

	clusterArns := listClusters.ClusterArns
	if len(clusterArns) == 0 {
		return nil, fmt.Errorf("no clusters found")
	}

	return clusterArns, nil
}

func (c *awsClient) DescribeClusters(
	ctx context.Context,
	clusterArns []string,
) ([]ecsTypes.Cluster, error) {
	describeClusters, err := c.ecsClient.DescribeClusters(ctx, &ecs.DescribeClustersInput{
		Clusters: clusterArns,
	})
	if err != nil {
		return nil, err
	}

	return describeClusters.Clusters, nil
}

func (c *awsClient) ListServices(ctx context.Context, clusterArn string) ([]string, error) {
	listServices, err := c.ecsClient.ListServices(ctx, &ecs.ListServicesInput{
		Cluster: &clusterArn,
	})
	if err != nil {
		return nil, err
	}

	serviceArns := listServices.ServiceArns
	if len(serviceArns) == 0 {
		return nil, fmt.Errorf("no services found in cluster %s", clusterArn)
	}

	return serviceArns, nil
}

func (c *awsClient) DescribeServices(
	ctx context.Context,
	clusterArn string,
	serviceArns []string,
) ([]ecsTypes.Service, error) {
	describeServices, err := c.ecsClient.DescribeServices(ctx, &ecs.DescribeServicesInput{
		Cluster:  &clusterArn,
		Services: serviceArns,
	})
	if err != nil {
		return nil, err
	}

	return describeServices.Services, nil
}

func (c *awsClient) ListTasks(
	ctx context.Context,
	clusterArn string,
	serviceName string,
) ([]string, error) {
	listTasks, err := c.ecsClient.ListTasks(ctx, &ecs.ListTasksInput{
		Cluster:     &clusterArn,
		ServiceName: &serviceName,
	})
	if err != nil {
		return nil, err
	}

	taskArns := listTasks.TaskArns
	if len(taskArns) == 0 {
		return nil, fmt.Errorf("no tasks found in service %s", serviceName)
	}

	return taskArns, nil
}

func (c *awsClient) DescribeTasks(
	ctx context.Context,
	clusterArn string,
	taskArns []string,
) ([]ecsTypes.Task, error) {
	describeTasks, err := c.ecsClient.DescribeTasks(ctx, &ecs.DescribeTasksInput{
		Cluster: &clusterArn,
		Tasks:   taskArns,
	})
	if err != nil {
		return nil, err
	}

	return describeTasks.Tasks, nil
}

func (c *awsClient) ExecuteCommand(
	ctx context.Context,
	cluster *ecsTypes.Cluster,
	taskArn string,
	container *ecsTypes.Container,
	command string,
	interactive bool,
) (*exec.Cmd, error) {
	smpPath, err := exec.LookPath("session-manager-plugin")
	if err != nil {
		return nil, err
	}

	executeCommand, err := c.ecsClient.ExecuteCommand(ctx, &ecs.ExecuteCommandInput{
		Cluster:     cluster.ClusterArn,
		Task:        &taskArn,
		Container:   container.Name,
		Command:     &command,
		Interactive: interactive,
	})
	if err != nil {
		return nil, err
	}

	session, err := json.Marshal(executeCommand.Session)
	if err != nil {
		return nil, err
	}

	taskArnSlices := strings.Split(taskArn, "/")
	if len(taskArnSlices) < 2 {
		return nil, fmt.Errorf("unable to extract task name from '%s'", taskArn)
	}

	taskName := strings.Join(taskArnSlices[1:], "/")
	target := fmt.Sprintf(
		"ecs:%s_%s_%s",
		*cluster.ClusterName,
		taskName,
		*container.RuntimeId,
	)
	startSessionInput, err := json.Marshal(ssm.StartSessionInput{
		Target: &target,
	})
	if err != nil {
		return nil, err
	}

	cmd := exec.Command(
		smpPath,
		string(session),
		c.region,
		"StartSession",
		"",
		string(startSessionInput),
		fmt.Sprintf("https://ssm.%s.amazonaws.com", c.region),
	)

	return cmd, nil
}

func (c *awsClient) ListTaskDefinitions(
	ctx context.Context,
	familyPrefix string,
) ([]string, error) {
	listTaskDefinitions, err := c.ecsClient.ListTaskDefinitions(
		ctx,
		&ecs.ListTaskDefinitionsInput{
			FamilyPrefix: &familyPrefix,
			Sort:         ecsTypes.SortOrderDesc,
		},
	)
	if err != nil {
		return nil, err
	}

	return listTaskDefinitions.TaskDefinitionArns, nil
}

func (c *awsClient) DescribeTaskDefinition(
	ctx context.Context,
	taskDefinitionArn string,
) (*ecsTypes.TaskDefinition, error) {
	describeTaskDefinition, err := c.ecsClient.DescribeTaskDefinition(
		ctx,
		&ecs.DescribeTaskDefinitionInput{
			TaskDefinition: &taskDefinitionArn,
		},
	)
	if err != nil {
		return nil, err
	}

	return describeTaskDefinition.TaskDefinition, nil
}

// CloudWatch Logs implementation

func (c *awsClient) StartLiveTail(
	ctx context.Context,
	logGroupName string,
	streamName string,
	handler LiveTailHandlers,
) error {
	// Describe log groups to get the ARN
	describeLogGroups, err := c.logsClient.DescribeLogGroups(ctx, &logs.DescribeLogGroupsInput{
		LogGroupNamePrefix: &logGroupName,
	})
	if err != nil {
		return err
	}

	logGroups := describeLogGroups.LogGroups
	if len(logGroups) == 0 {
		return fmt.Errorf("no log group '%s' found", logGroupName)
	}

	// Start the live tail
	startLiveTail, err := c.logsClient.StartLiveTail(ctx, &logs.StartLiveTailInput{
		LogGroupIdentifiers: []string{*logGroups[0].LogGroupArn},
		LogStreamNames:      []string{streamName},
	})
	if err != nil {
		return fmt.Errorf("failed to start live tail: %w", err)
	}

	// Get the stream
	stream := startLiveTail.GetStream()
	defer func() {
		if err = stream.Close(); err != nil {
			log.Printf("Unable to close stream: %v", err)
		}
	}()

	eventsStream := stream.Events()

	// Process events
	for {
		event := <-eventsStream
		switch e := event.(type) {
		case *logsTypes.StartLiveTailResponseStreamMemberSessionStart:
			handler.Start()
		case *logsTypes.StartLiveTailResponseStreamMemberSessionUpdate:
			for _, result := range e.Value.SessionResults {
				handler.Update(result)
			}
		default:
			if err := stream.Err(); err != nil {
				return err
			}
			if event == nil {
				return fmt.Errorf("stream is closed")
			}
			return fmt.Errorf("unknown event type: %T", e)
		}
	}
}

func (c *awsClient) UpdateService(
	ctx context.Context,
	service ecsTypes.Service,
	config ServiceConfig,
	waitTimeout time.Duration,
) (*ecsTypes.Service, error) {
	updateService, err := c.ecsClient.UpdateService(ctx, &ecs.UpdateServiceInput{
		Service:        service.ServiceArn,
		TaskDefinition: &config.TaskDefinitionArn,
		DesiredCount:   &config.DesiredCount,
	})
	if err != nil {
		return nil, err
	}

	waiter := ecs.NewServicesStableWaiter(c.ecsClient)
	err = waiter.Wait(ctx, &ecs.DescribeServicesInput{
		Cluster:  updateService.Service.ClusterArn,
		Services: []string{*updateService.Service.ServiceArn},
	}, waitTimeout)
	if err != nil {
		return nil, err
	}

	return updateService.Service, nil
}
