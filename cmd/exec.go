package cmd

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"os"
	"os/exec"
	"strings"

	"github.com/aws/aws-sdk-go-v2/config"
	"github.com/aws/aws-sdk-go-v2/service/ecs"
	"github.com/aws/aws-sdk-go-v2/service/ecs/types"
	"github.com/aws/aws-sdk-go-v2/service/ssm"
	"github.com/pterm/pterm"
	"github.com/spf13/cobra"
)

var execCmd = &cobra.Command{
	Use:   "exec",
	Short: "Runs a command remotely on a container",
	RunE: func(cmd *cobra.Command, args []string) error {
		clusterId, err := cmd.Flags().GetString("cluster")
		if err != nil {
			return fmt.Errorf("Unable to parse 'cluster' flag: %w", err)
		}
		taskId, err := cmd.Flags().GetString("task")
		if err != nil {
			return fmt.Errorf("Unable to parse 'task' flag: %w", err)
		}
		containerId, err := cmd.Flags().GetString("container")
		if err != nil {
			return fmt.Errorf("Unable to parse 'container' flag: %w", err)
		}
		command, err := cmd.Flags().GetString("command")
		if err != nil {
			return fmt.Errorf("Unable to parse 'command' flag: %w", err)
		}
		interactive, err := cmd.Flags().GetBool("interactive")
		if err != nil {
			return fmt.Errorf("Unable to parse 'interactive' flag: %w", err)
		}
		return runExec(context.TODO(), clusterId, taskId, containerId, command, interactive)
	},
}

func runExec(ctx context.Context, clusterId string, taskId string, containerId string, command string, interactive bool) error {
	cfg, err := config.LoadDefaultConfig(ctx)
	if err != nil {
		return fmt.Errorf("Error loading AWS configuration: %w", err)
	}
	client := ecs.NewFromConfig(cfg)
	cluster, err := selectCluster(ctx, client, clusterId)
	if err != nil {
		return err
	}
	task, err := selectTask(ctx, client, *cluster.ClusterName, taskId)
	if err != nil {
		return err
	}
	container, err := selectContainer(*task, containerId)
	if err != nil {
		return err
	}
	output, err := client.ExecuteCommand(ctx, &ecs.ExecuteCommandInput{
		Cluster:     cluster.ClusterArn,
		Task:        task.TaskArn,
		Container:   container.Name,
		Command:     &command,
		Interactive: interactive,
	})
	if err != nil {
		return err
	}
	session, err := json.Marshal(output.Session)
	if err != nil {
		return err
	}
	taskArnSlices := strings.Split(*task.TaskArn, "/")
	if len(taskArnSlices) < 2 {
		return fmt.Errorf("Unable to extract task name from '%s'", *task.TaskArn)
	}
	taskName := strings.Join(taskArnSlices[1:], "/")
	var target = fmt.Sprintf("ecs:%s_%s_%s", *cluster.ClusterName, taskName, *container.RuntimeId)
	targetJSON, err := json.Marshal(ssm.StartSessionInput{
		Target: &target,
	})
	if err != nil {
		return err
	}
	pterm.Info.Printfln("interactive-ecs exec --cluster %s --task %s --container %s --command \"%s\" --interactive %t\n",
		*cluster.ClusterName,
		taskName,
		*container.Name,
		command,
		interactive,
	)
	// https://github.com/aws/aws-cli/blob/develop/awscli/customizations/ecs/executecommand.py
	var smpArgs = []string{
		string(session),
		cfg.Region,
		"StartSession",
		"",
		string(targetJSON),
		fmt.Sprintf("https://ssm.%s.amazonaws.com", cfg.Region),
	}
	smp := exec.Command("session-manager-plugin", smpArgs...)
	smp.Stdin = os.Stdin
	smp.Stdout = os.Stdout
	smp.Stderr = os.Stderr
	return smp.Run()
}

func selectTask(ctx context.Context, client *ecs.Client, clusterId string, taskId string) (*types.Task, error) {
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

func selectContainer(task types.Task, containerId string) (*types.Container, error) {
	var containerNames []string
	for _, container := range task.Containers {
		if *container.Name == containerId {
			return &container, nil
		}
		containerNames = append(containerNames, *container.Name)
	}
	containerName, err := pterm.DefaultInteractiveSelect.WithOptions(containerNames).Show("Select a container")
	if err != nil {
		return nil, fmt.Errorf("Error selecting a container: %w", err)
	}
	for _, container := range task.Containers {
		if *container.Name == containerName {
			return &container, nil
		}
	}
	return nil, fmt.Errorf("No container '%s' found in task '%s'", containerName, *task.TaskArn)
}

func init() {
	rootCmd.AddCommand(execCmd)

	execCmd.Flags().StringP("cluster", "c", "", "The cluster ARN or name")
	execCmd.Flags().StringP("task", "t", "", "The task ARN or name")
	execCmd.Flags().StringP("container", "n", "", "The container ARN or name")
	execCmd.Flags().StringP("command", "m", "/bin/sh", "The command to run on the container")
	execCmd.Flags().BoolP("interactive", "i", true, "Runs command on interactive mode")
}
