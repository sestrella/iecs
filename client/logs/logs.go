package logs

import (
	"context"
	"fmt"
	"log"
	"time"

	"github.com/aws/aws-sdk-go-v2/aws"
	"github.com/aws/aws-sdk-go-v2/service/cloudwatchlogs"
	"github.com/aws/aws-sdk-go-v2/service/cloudwatchlogs/types"
)

// EventHandler is a function that handles log events.
type EventHandler func(timestamp time.Time, message string)

// Client interface for CloudWatch Logs operations.
type Client interface {
	// DescribeLogGroups describes log groups with the given prefix.
	DescribeLogGroups(
		ctx context.Context,
		logGroupNamePrefix string,
	) (*cloudwatchlogs.DescribeLogGroupsOutput, error)

	// StartLiveTail starts streaming logs from a log group with a given prefix.
	StartLiveTail(
		ctx context.Context,
		logGroupName string,
		streamPrefix string,
		handler EventHandler,
	) error
}

// awsClient implements the Client interface for CloudWatch Logs.
type awsClient struct {
	client *cloudwatchlogs.Client
}

// NewClient creates a new CloudWatch Logs client.
func NewClient(cfg aws.Config) Client {
	client := cloudwatchlogs.NewFromConfig(cfg)
	return &awsClient{
		client: client,
	}
}

// DescribeLogGroups describes log groups with the given prefix.
func (c *awsClient) DescribeLogGroups(
	ctx context.Context,
	logGroupNamePrefix string,
) (*cloudwatchlogs.DescribeLogGroupsOutput, error) {
	input := &cloudwatchlogs.DescribeLogGroupsInput{}
	if logGroupNamePrefix != "" {
		input.LogGroupNamePrefix = &logGroupNamePrefix
	}

	output, err := c.client.DescribeLogGroups(ctx, input)
	if err != nil {
		return nil, fmt.Errorf("failed to describe log groups: %w", err)
	}

	if len(output.LogGroups) == 0 {
		return nil, fmt.Errorf("no log groups found with prefix: %s", logGroupNamePrefix)
	}

	return output, nil
}

// StartLiveTail starts streaming logs from a log group with the given prefix.
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
	startLiveTail, err := c.client.StartLiveTail(
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
	defer stream.Close()

	eventsChannel := stream.Events()

	// Process events
	for {
		select {
		case <-ctx.Done():
			log.Println("Context cancelled, stopping event handler.")
			return nil
		case event := <-eventsChannel:
			switch e := event.(type) {
			case *types.StartLiveTailResponseStreamMemberSessionStart:
				log.Printf("Live Tail Session Started: RequestId: %s, SessionId: %s\n", *e.Value.RequestId, *e.Value.SessionId)
			case *types.StartLiveTailResponseStreamMemberSessionUpdate:
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
