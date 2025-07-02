package client

import (
	"context"
	"encoding/json"
	"fmt"
	"io"
	"log"
	"os"
	"os/exec"
	"os/signal"
	"strings"
	"time"

	"github.com/aws/aws-sdk-go-v2/aws"
	logs "github.com/aws/aws-sdk-go-v2/service/cloudwatchlogs"
	logsTypes "github.com/aws/aws-sdk-go-v2/service/cloudwatchlogs/types"
	"github.com/aws/aws-sdk-go-v2/service/ecs"
	ecsTypes "github.com/aws/aws-sdk-go-v2/service/ecs/types"
	"github.com/aws/aws-sdk-go-v2/service/ssm"
)

// EventHandler is a function that handles log events.
type EventHandler func(timestamp time.Time, message string)

// Client interface combines ECS and CloudWatch Logs operations.
type Client interface {
	DescribeClusters(ctx context.Context) ([]ecsTypes.Cluster, error)
	DescribeServices(ctx context.Context, clusterArn string) ([]ecsTypes.Service, error)
	ExecuteCommand2(
		ctx context.Context,
		cluster *ecsTypes.Cluster,
		task *ecsTypes.Task,
		container *ecsTypes.Container,
		command string,
	) error

	// CloudWatch Logs operations
	StartLiveTail(
		ctx context.Context,
		logGroupName string,
		streamPrefix string,
		handler EventHandler,
	) error

	// ECS operations
	ExecuteCommand(
		ctx context.Context,
		cluster string,
		task string,
		container string,
		command string,
		interactive bool,
	) (*ecs.ExecuteCommandOutput, error)
}

// awsClient implements the combined Client interface
type awsClient struct {
	cfg        aws.Config
	ecsClient  *ecs.Client
	logsClient *logs.Client
}

// NewClient creates a new combined AWS client
func NewClient(cfg aws.Config) Client {
	ecsClient := ecs.NewFromConfig(cfg)
	logsClient := logs.NewFromConfig(cfg)
	return &awsClient{
		cfg:        cfg,
		ecsClient:  ecsClient,
		logsClient: logsClient,
	}
}

// ECS operations implementation

func (c *awsClient) DescribeClusters(ctx context.Context) ([]ecsTypes.Cluster, error) {
	listedClusters, err := c.ecsClient.ListClusters(ctx, &ecs.ListClustersInput{})
	if err != nil {
		return nil, err
	}
	describedClusters, err := c.ecsClient.DescribeClusters(ctx, &ecs.DescribeClustersInput{
		Clusters: listedClusters.ClusterArns,
	})
	if err != nil {
		return nil, err
	}
	return describedClusters.Clusters, nil
}

func (c *awsClient) DescribeServices(
	ctx context.Context,
	clusterArn string,
) ([]ecsTypes.Service, error) {
	listedServices, err := c.ecsClient.ListServices(ctx, &ecs.ListServicesInput{
		Cluster: &clusterArn,
	})
	if err != nil {
		return nil, err
	}
	describedServices, err := c.ecsClient.DescribeServices(ctx, &ecs.DescribeServicesInput{
		Cluster:  &clusterArn,
		Services: listedServices.ServiceArns,
	})
	if err != nil {
		return nil, err
	}
	return describedServices.Services, nil
}

func (c *awsClient) ExecuteCommand(
	ctx context.Context,
	cluster string,
	task string,
	container string,
	command string,
	interactive bool,
) (*ecs.ExecuteCommandOutput, error) {
	return c.ecsClient.ExecuteCommand(ctx, &ecs.ExecuteCommandInput{
		Cluster:     &cluster,
		Task:        &task,
		Container:   &container,
		Command:     &command,
		Interactive: interactive,
	})
}

// CloudWatch Logs implementation

func (c *awsClient) describeLogGroups(
	ctx context.Context,
	logGroupNamePrefix string,
) (*logs.DescribeLogGroupsOutput, error) {
	input := &logs.DescribeLogGroupsInput{}
	if logGroupNamePrefix != "" {
		input.LogGroupNamePrefix = &logGroupNamePrefix
	}

	output, err := c.logsClient.DescribeLogGroups(ctx, input)
	if err != nil {
		return nil, fmt.Errorf("failed to describe log groups: %w", err)
	}

	if len(output.LogGroups) == 0 {
		return nil, fmt.Errorf("no log groups found with prefix: %s", logGroupNamePrefix)
	}

	return output, nil
}

func (c *awsClient) StartLiveTail(
	ctx context.Context,
	logGroupName string,
	streamPrefix string,
	handler EventHandler,
) error {
	// Describe log groups to get the ARN
	describeOutput, err := c.describeLogGroups(ctx, logGroupName)
	if err != nil {

		logGroupArn := describeOutput.LogGroups[0].LogGroupArn
		if logGroupArn == nil {
			return fmt.Errorf("log group ARN is nil for group: %s", logGroupName)
		}

		// Start the live tail
		startLiveTail, err := c.logsClient.StartLiveTail(
			ctx,
			&logs.StartLiveTailInput{
				LogGroupIdentifiers:   []string{*logGroupArn},
				LogStreamNamePrefixes: []string{streamPrefix},
			},
		)
		if err != nil {
			return fmt.Errorf("failed to start live tail: %w", err)
		}

		// Get the stream
		stream := startLiveTail.GetStream()
		defer func() {
			if err = stream.Close(); err != nil {
				log.Fatal(err)
			}
		}()

		eventsChannel := stream.Events()

		// Process events
		for {
			select {
			case <-ctx.Done():
				log.Println("Context cancelled, stopping event handler.")
				return nil
			case event := <-eventsChannel:
				switch e := event.(type) {
				case *logsTypes.StartLiveTailResponseStreamMemberSessionStart:
					log.Printf("Live Tail Session Started: RequestId: %s, SessionId: %s\n", *e.Value.RequestId, *e.Value.SessionId)
				case *logsTypes.StartLiveTailResponseStreamMemberSessionUpdate:
					for _, logEvent := range e.Value.SessionResults {
						date := time.UnixMilli(*logEvent.Timestamp)
						handler(date, *logEvent.Message)
					}
				default:
					log.Printf("Received unknown event type: %T\n", e)
					if err := stream.Err(); err != nil {
						return err
					}
				}
			}
		}
	}
	return nil
}

func (c *awsClient) ExecuteCommand2(
	ctx context.Context,
	cluster *ecsTypes.Cluster,
	task *ecsTypes.Task,
	container *ecsTypes.Container,
	command string,
) error {
	executeCommand, err := c.ecsClient.ExecuteCommand(ctx, &ecs.ExecuteCommandInput{
		Cluster:     cluster.ClusterArn,
		Task:        task.TaskArn,
		Container:   container.Name,
		Command:     &command,
		Interactive: true,
	})
	if err != nil {
		return err
	}

	session, err := json.Marshal(executeCommand.Session)
	if err != nil {
		return err
	}

	taskArnSlices := strings.Split(*task.TaskArn, "/")
	if len(taskArnSlices) < 2 {
		// TODO: review error message
		return fmt.Errorf("unable to extract task name from '%s'", *task.TaskArn)
	}

	taskName := strings.Join(taskArnSlices[1:], "/")
	target := fmt.Sprintf(
		"ecs:%s_%s_%s",
		*cluster.ClusterName,
		taskName,
		*container.RuntimeId,
	)
	startSession, err := json.Marshal(ssm.StartSessionInput{
		Target: &target,
	})
	if err != nil {
		return err
	}

	region := c.cfg.Region
	cmd := exec.Command(
		"session-manager-plugin",
		string(session),
		region,
		"StartSession",
		"",
		string(startSession),
		fmt.Sprintf("https://ssm.%s.amazonaws.com", region),
	)
	cmd.Stdin = os.Stdin
	cmd.Stdout = os.Stdout
	cmd.Stderr = os.Stderr

	stop := make(chan os.Signal, 1)
	defer signal.Stop(stop)

	go func() {
		signal.Notify(stop, os.Interrupt)
		<-stop

		if err := cmd.Process.Kill(); err != nil {
			// gui.Log.Error(err)
		}
	}()

	if err := cmd.Run(); err != nil {
		// gui.Log.Error(err)
	}

	cmd.Stdin = nil
	cmd.Stdout = io.Discard
	cmd.Stderr = io.Discard

	return nil
}
