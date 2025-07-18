//go:build DEMO

package client

import (
	"context"
	"fmt"
	"os/exec"
	"time"

	"github.com/aws/aws-sdk-go-v2/aws"
	logsTypes "github.com/aws/aws-sdk-go-v2/service/cloudwatchlogs/types"
	ecsTypes "github.com/aws/aws-sdk-go-v2/service/ecs/types"
)

var _ Client = DemoClient{}

type DemoClient struct{}

func NewClient(_ aws.Config) Client {
	return &DemoClient{}
}

func (c DemoClient) ListClusters(ctx context.Context) ([]string, error) {
	return []string{
		"arn:aws:ecs:us-east-1:123456789012:cluster/cluster-1",
		"arn:aws:ecs:us-east-1:123456789012:cluster/cluster-2",
	}, nil
}

func (c DemoClient) DescribeClusters(
	ctx context.Context,
	clusterArns []string,
) ([]ecsTypes.Cluster, error) {
	clusters := []ecsTypes.Cluster{}
	for _, arn := range clusterArns {
		clusters = append(clusters, ecsTypes.Cluster{
			ClusterArn:  aws.String(arn),
			ClusterName: aws.String(fmt.Sprintf("cluster-%d", len(clusters)+1)),
			Status:      aws.String("ACTIVE"),
		})
	}
	return clusters, nil
}

func (c DemoClient) ListServices(ctx context.Context, clusterArn string) ([]string, error) {
	return []string{
		"arn:aws:ecs:us-east-1:123456789012:service/cluster-1/service-1",
		"arn:aws:ecs:us-east-1:123456789012:service/cluster-1/service-2",
	}, nil
}

func (c DemoClient) DescribeServices(
	ctx context.Context,
	clusterArn string,
	serviceArns []string,
) ([]ecsTypes.Service, error) {
	services := []ecsTypes.Service{}
	for _, arn := range serviceArns {
		services = append(services, ecsTypes.Service{
			ServiceArn:  aws.String(arn),
			ServiceName: aws.String(fmt.Sprintf("service-%d", len(services)+1)),
			ClusterArn:  aws.String(clusterArn),
			Status:      aws.String("ACTIVE"),
			TaskDefinition: aws.String(
				"arn:aws:ecs:us-east-1:123456789012:task-definition/task-def-1:1",
			),
		})
	}
	return services, nil
}

func (c DemoClient) UpdateService(
	ctx context.Context,
	service *ecsTypes.Service,
	input ServiceConfig,
	waitTimeout time.Duration,
) (*ecsTypes.Service, error) {
	return nil, nil
}

func (c DemoClient) ListTasks(
	ctx context.Context,
	clusterArn string,
	serviceArn string,
) ([]string, error) {
	return []string{
		"arn:aws:ecs:us-east-1:123456789012:task/cluster-1/task-1",
		"arn:aws:ecs:us-east-1:123456789012:task/cluster-1/task-2",
	}, nil
}

func (c DemoClient) DescribeTasks(
	ctx context.Context,
	clusterArn string,
	taskArns []string,
) ([]ecsTypes.Task, error) {
	tasks := []ecsTypes.Task{}
	for _, arn := range taskArns {
		tasks = append(tasks, ecsTypes.Task{
			TaskArn:       aws.String(arn),
			ClusterArn:    aws.String(clusterArn),
			LastStatus:    aws.String("RUNNING"),
			DesiredStatus: aws.String("RUNNING"),
			TaskDefinitionArn: aws.String(
				"arn:aws:ecs:us-east-1:123456789012:task-definition/task-def-1:1",
			),
			Containers: []ecsTypes.Container{
				{
					Name:      aws.String("container-1"),
					RuntimeId: aws.String("runtime-id-1"),
				},
				{
					Name:      aws.String("container-2"),
					RuntimeId: aws.String("runtime-id-2"),
				},
			},
		})
	}
	return tasks, nil
}

func (c DemoClient) ListTaskDefinitions(
	ctx context.Context,
	familyPrefix string,
) ([]string, error) {
	return nil, nil
}

func (c DemoClient) DescribeTaskDefinition(
	ctx context.Context,
	taskDefinitionArn string,
) (*ecsTypes.TaskDefinition, error) {
	return &ecsTypes.TaskDefinition{
		TaskDefinitionArn: aws.String(taskDefinitionArn),
		Family:            aws.String("task-def-1"),
		Revision:          1,
		ContainerDefinitions: []ecsTypes.ContainerDefinition{
			{
				Name: aws.String("container-1"),
				LogConfiguration: &ecsTypes.LogConfiguration{
					LogDriver: ecsTypes.LogDriverAwslogs,
					Options: map[string]string{
						"awslogs-group":         "log-group-1",
						"awslogs-region":        "us-east-1",
						"awslogs-stream-prefix": "prefix-1",
					},
				},
			},
			{
				Name: aws.String("container-2"),
				LogConfiguration: &ecsTypes.LogConfiguration{
					LogDriver: ecsTypes.LogDriverAwslogs,
					Options: map[string]string{
						"awslogs-group":         "log-group-2",
						"awslogs-region":        "us-east-1",
						"awslogs-stream-prefix": "prefix-2",
					},
				},
			},
		},
	}, nil
}

func (c DemoClient) StartLiveTail(
	ctx context.Context,
	logGroupName string,
	streamPrefix string,
	handler LiveTailHandlers,
) error {
	handler.Start()
	for i := range 5 {
		handler.Update(logsTypes.LiveTailSessionLogEvent{
			Message:   aws.String(fmt.Sprintf("log message %d", i)),
			Timestamp: aws.Int64(time.Now().UnixMilli()),
		})
		time.Sleep(100 * time.Millisecond)
	}
	return nil
}

func (c DemoClient) ExecuteCommand(
	ctx context.Context,
	cluster *ecsTypes.Cluster,
	taskArn string,
	container *ecsTypes.Container,
	command string,
	interactive bool,
) (*exec.Cmd, error) {
	cmd := exec.Command(command)
	return cmd, nil
}
