package selector

import (
	"context"
	"errors"
	"fmt"

	"github.com/aws/aws-sdk-go-v2/service/ecs"
	"github.com/aws/aws-sdk-go-v2/service/ecs/types"
	"github.com/pterm/pterm"
)

func SelectTask(ctx context.Context, client *ecs.Client, clusterId string, taskId string) (*types.Task, error) {
	if taskId != "" {
		return describeTask(ctx, client, clusterId, taskId)
	}
	output, err := client.ListTasks(ctx, &ecs.ListTasksInput{
		Cluster: &clusterId,
	})
	if err != nil {
		return nil, fmt.Errorf("Error listing tasks: %w", err)
	}
	if len(output.TaskArns) == 0 {
		return nil, errors.New("No tasks found")
	}
	taskArn, err := pterm.DefaultInteractiveSelect.WithOptions(output.TaskArns).Show("Select a task")
	if err != nil {
		return nil, fmt.Errorf("Error selecting task: %w", err)
	}
	return describeTask(ctx, client, clusterId, taskArn)
}

func describeTask(ctx context.Context, client *ecs.Client, clusterId string, taskId string) (*types.Task, error) {
	output, err := client.DescribeTasks(ctx, &ecs.DescribeTasksInput{
		Cluster: &clusterId,
		Tasks:   []string{taskId},
	})
	if err != nil {
		return nil, fmt.Errorf("Error describing task '%s': %w", taskId, err)
	}
	if len(output.Tasks) == 0 {
		return nil, fmt.Errorf("No task '%s' found", taskId)
	}
	return &output.Tasks[0], nil
}
