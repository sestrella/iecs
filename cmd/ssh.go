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

// sshCmd represents the exec command
var sshCmd = &cobra.Command{
	Use:   "ssh",
	Short: "A brief description of your command",
	Long:  "TODO",
	RunE: func(cmd *cobra.Command, args []string) error {
		clusterId, _ := cmd.Flags().GetString("cluster")
		taskId, _ := cmd.Flags().GetString("task")
		containerId, _ := cmd.Flags().GetString("container")
		command, _ := cmd.Flags().GetString("command")
		interactive, _ := cmd.Flags().GetBool("interactive")
		return runCommand(context.TODO(), clusterId, taskId, containerId, command, interactive)
	},
}

func runCommand(ctx context.Context, clusterId string, taskId string, containerId string, command string, interactive bool) error {
	cfg, err := config.LoadDefaultConfig(ctx)
	if err != nil {
		return err
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
	container, err := selectContainer(ctx, client, *task, containerId)
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
	var target = fmt.Sprintf("ecs:%s_%s_%s", *cluster.ClusterName, taskArnSlices[0], *container.RuntimeId)
	targetJSON, err := json.Marshal(ssm.StartSessionInput{
		Target: &target,
	})
	if err != nil {
		return err
	}
	fmt.Print("\nNon-interactive command:\n")
	fmt.Printf("\n\tlazy-ecs ssh --cluster %s --task %s --container %s --command \"%s\" --interactive %t\n",
		*cluster.ClusterName,
		*task.TaskArn,
		*container.Name,
		command,
		interactive,
	)
	// https://github.com/aws/aws-cli/blob/develop/awscli/customizations/ecs/executecommand.py
	var argsFoo = []string{string(session),
		cfg.Region,
		"StartSession",
		"",
		string(targetJSON),
		fmt.Sprintf("https://ssm.%s.amazonaws.com", cfg.Region),
	}
	fmt.Println(argsFoo)
	smp := exec.Command("session-manager-plugin", argsFoo...)
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

func selectContainer(ctx context.Context, client *ecs.Client, task types.Task, containerId string) (*types.Container, error) {
	var containerNames []string
	for _, container := range task.Containers {
		if container.Name == &containerId {
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
	rootCmd.AddCommand(sshCmd)

	sshCmd.Flags().StringP("cluster", "c", "", "TODO")
	sshCmd.Flags().StringP("task", "t", "", "TODO")
	sshCmd.Flags().StringP("container", "n", "", "TODO")
	sshCmd.Flags().StringP("command", "m", "/bin/sh", "TODO")
	sshCmd.Flags().BoolP("interactive", "i", true, "TODO")
}
