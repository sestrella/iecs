package client

import (
	"context"
	"fmt"
	"log"
	"time"

	"github.com/aws/aws-sdk-go-v2/aws"
	"github.com/aws/aws-sdk-go-v2/service/cloudwatchlogs"
	logsTypes "github.com/aws/aws-sdk-go-v2/service/cloudwatchlogs/types"
	"github.com/aws/aws-sdk-go-v2/service/ecs"
)

// EventHandler is a function that handles log events.
type EventHandler func(timestamp time.Time, message string)

// Client interface combines ECS and CloudWatch Logs operations.
type Client interface {
	// ECS operations
	ExecuteCommand(
		ctx context.Context,
		clusterArn *string,
		taskArn *string,
		containerName *string,
		command string,
		interactive bool,
	) (*ecs.ExecuteCommandOutput, error)

	// CloudWatch Logs operations
	DescribeLogGroups(
		ctx context.Context,
		logGroupNamePrefix string,
	) (*cloudwatchlogs.DescribeLogGroupsOutput, error)
	StartLiveTail(
		ctx context.Context,
		logGroupName string,
		streamPrefix string,
		handler EventHandler,
	) error
}

// awsClient implements the combined Client interface
type awsClient struct {
	ecsClient  *ecs.Client
	logsClient *cloudwatchlogs.Client
}

// NewClient creates a new combined AWS client
func NewClient(cfg aws.Config) Client {
	ecsClient := ecs.NewFromConfig(cfg)
	logsClient := cloudwatchlogs.NewFromConfig(cfg)
	return &awsClient{
		ecsClient:  ecsClient,
		logsClient: logsClient,
	}
}

func (c *awsClient) ExecuteCommand(
	ctx context.Context,
	clusterArn *string,
	taskArn *string,
	containerName *string,
	command string,
	interactive bool,
) (*ecs.ExecuteCommandOutput, error) {
	input := &ecs.ExecuteCommandInput{
		Cluster:     clusterArn,
		Task:        taskArn,
		Container:   containerName,
		Command:     &command,
		Interactive: interactive,
	}

	output, err := c.ecsClient.ExecuteCommand(ctx, input)
	if err != nil {
		return nil, fmt.Errorf("failed to execute command: %w", err)
	}

	return output, nil
}

// CloudWatch Logs implementation

func (c *awsClient) DescribeLogGroups(
	ctx context.Context,
	logGroupNamePrefix string,
) (*cloudwatchlogs.DescribeLogGroupsOutput, error) {
	input := &cloudwatchlogs.DescribeLogGroupsInput{}
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
	describeOutput, err := c.DescribeLogGroups(ctx, logGroupName)
	if err != nil {
		return err
	}

	logGroupArn := describeOutput.LogGroups[0].LogGroupArn
	if logGroupArn == nil {
		return fmt.Errorf("log group ARN is nil for group: %s", logGroupName)
	}

	// Start the live tail
	startLiveTail, err := c.logsClient.StartLiveTail(
		ctx,
		&cloudwatchlogs.StartLiveTailInput{
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
