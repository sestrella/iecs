package client

import (
	"context"
	"fmt"
	"log"

	"github.com/aws/aws-sdk-go-v2/aws"
	logs "github.com/aws/aws-sdk-go-v2/service/cloudwatchlogs"
	logsTypes "github.com/aws/aws-sdk-go-v2/service/cloudwatchlogs/types"
	"github.com/aws/aws-sdk-go-v2/service/ecs"
)

// EventHandler is a function that handles log events.
type LiveTailHandlers struct {
	Start  func()
	Update func(logsTypes.LiveTailSessionLogEvent)
}

// Client interface combines ECS and CloudWatch Logs operations.
type Client interface {
	// CloudWatch Logs operations
	StartLiveTail(
		ctx context.Context,
		logGroupName string,
		streamPrefix string,
		handler LiveTailHandlers,
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
	ecsClient  *ecs.Client
	logsClient *logs.Client
}

// NewClient creates a new combined AWS client
func NewClient(cfg aws.Config) Client {
	ecsClient := ecs.NewFromConfig(cfg)
	logsClient := logs.NewFromConfig(cfg)
	return &awsClient{
		ecsClient:  ecsClient,
		logsClient: logsClient,
	}
}

// ECS operations implementation

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
