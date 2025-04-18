package selector

import (
	"context"
	"fmt"

	"github.com/aws/aws-sdk-go-v2/service/ecs/types"
	"github.com/charmbracelet/huh"
	"github.com/sestrella/iecs/client"
)

func TaskSelector(
	ctx context.Context,
	client client.Client,
	clusterArn string,
	serviceArn string,
) (*types.Task, error) {
	taskArns, err := client.ListTasks(ctx, clusterArn, serviceArn)
	if err != nil {
		return nil, err
	}

	var selectedTaskArn string
	if len(taskArns) == 1 {
		selectedTaskArn = taskArns[0]
	} else {
		form := huh.NewForm(
			huh.NewGroup(
				huh.NewSelect[string]().
					Title("Select task").
					Options(huh.NewOptions(taskArns...)...).
					Value(&selectedTaskArn).
					WithHeight(5),
			),
		)

		if err = form.Run(); err != nil {
			return nil, err
		}
	}

	task, err := client.DescribeTask(ctx, clusterArn, selectedTaskArn)
	if err != nil {
		return nil, err
	}

	fmt.Printf("Selected task: %s\n", *task.TaskArn)
	return task, nil
}
