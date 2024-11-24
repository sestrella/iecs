package selector

import (
	"context"
	"fmt"

	"github.com/aws/aws-sdk-go-v2/service/ecs"
	"github.com/aws/aws-sdk-go-v2/service/ecs/types"
	"github.com/pterm/pterm"
)

func SelectTask(ctx context.Context, client *ecs.Client, clusterId string, serviceId string, taskId string) (*types.Task, error) {
	if taskId == "" {
		listTasks, err := client.ListTasks(ctx, &ecs.ListTasksInput{
			Cluster:     &clusterId,
			ServiceName: &serviceId,
		})
		if err != nil {
			return nil, err
		}
		taskArn, err := pterm.DefaultInteractiveSelect.WithOptions(listTasks.TaskArns).Show("Task")
		if err != nil {
			return nil, err
		}
		taskId = taskArn
	}
	describeTasks, err := client.DescribeTasks(ctx, &ecs.DescribeTasksInput{
		Cluster: &clusterId,
		Tasks:   []string{taskId},
	})
	if err != nil {
		return nil, err
	}
	if len(describeTasks.Tasks) > 0 {
		return &describeTasks.Tasks[0], nil
	}
	return nil, fmt.Errorf("no task '%v' found", taskId)
}
