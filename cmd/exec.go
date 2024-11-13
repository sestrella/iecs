package cmd

import (
	"context"
	"encoding/json"
	"fmt"
	"log"
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

const EXEC_CLUSTER_FLAG = "cluster"
const EXEC_TASK_FLAG = "task"
const EXEC_CONTAINER_FLAG = "container"

var execCmd = &cobra.Command{
	Use:   "exec",
	Short: "Runs a command remotely on a container",
	Example: `
  aws-vault exec <profile> -- iecs exec -c /bin/bash (recommended)
  env AWS_PROFILE=<profile> iecs exec -c /bin/bash
  `,
	Run: func(cmd *cobra.Command, args []string) {
		clusterId, err := cmd.Flags().GetString(EXEC_CLUSTER_FLAG)
		if err != nil {
			log.Fatal(err)
		}
		taskId, err := cmd.Flags().GetString(EXEC_TASK_FLAG)
		if err != nil {
			log.Fatal(err)
		}
		containerId, err := cmd.Flags().GetString(EXEC_CONTAINER_FLAG)
		if err != nil {
			log.Fatal(err)
		}
		command, err := cmd.Flags().GetString("command")
		if err != nil {
			log.Fatal(err)
		}
		interactive, err := cmd.Flags().GetBool("interactive")
		if err != nil {
			log.Fatal(err)
		}
		cfg, err := config.LoadDefaultConfig(context.TODO())
		if err != nil {
			log.Fatal(err)
		}
		client := ecs.NewFromConfig(cfg)
		err = runExec(context.TODO(), client, cfg.Region, clusterId, taskId, containerId, command, interactive)
		if err != nil {
			log.Fatal(err)
		}
	},
}

func runExec(ctx context.Context, client *ecs.Client, region string, clusterId string, taskId string, containerId string, command string, interactive bool) error {
	cluster, err := describeCluster(ctx, client, clusterId)
	if err != nil {
		return err
	}
	task, err := describeTask(ctx, client, *cluster.ClusterArn, taskId)
	if err != nil {
		return err
	}
	container, err := describeContainer(task.Containers, containerId)
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
		return fmt.Errorf("Error executing command: %w", err)
	}
	session, err := json.Marshal(output.Session)
	if err != nil {
		return fmt.Errorf("Unable to encode session: %w", err)
	}
	taskArnSlices := strings.Split(*task.TaskArn, "/")
	if len(taskArnSlices) < 2 {
		return fmt.Errorf("Unable to extract task name from '%s'", *task.TaskArn)
	}
	taskName := strings.Join(taskArnSlices[1:], "/")
	target := fmt.Sprintf("ecs:%s_%s_%s", *cluster.ClusterName, taskName, *container.RuntimeId)
	targetJSON, err := json.Marshal(ssm.StartSessionInput{
		Target: &target,
	})
	if err != nil {
		return err
	}
	pterm.Info.Printfln("iecs exec --cluster %s --task %s --container %s --command \"%s\" --interactive %t\n",
		*cluster.ClusterName,
		taskName,
		*container.Name,
		command,
		interactive,
	)
	// https://github.com/aws/aws-cli/blob/develop/awscli/customizations/ecs/executecommand.py
	smp := exec.Command("session-manager-plugin",
		string(session),
		region,
		"StartSession",
		"",
		string(targetJSON),
		fmt.Sprintf("https://ssm.%s.amazonaws.com", region),
	)
	smp.Stdin = os.Stdin
	smp.Stdout = os.Stdout
	smp.Stderr = os.Stderr
	return smp.Run()
}

func describeContainer(containers []types.Container, containerId string) (*types.Container, error) {
	if containerId == "" {
		var containerNames []string
		for _, container := range containers {
			containerNames = append(containerNames, *container.Name)
		}
		containerName, err := pterm.DefaultInteractiveSelect.WithOptions(containerNames).Show("Container")
		if err != nil {
			return nil, err
		}
		containerId = containerName
	}
	for _, container := range containers {
		if *container.Name == containerId {
			return &container, nil
		}
	}
	return nil, fmt.Errorf("no container '%v' found", containerId)
}

func init() {
	rootCmd.AddCommand(execCmd)

	execCmd.Flags().StringP(EXEC_CLUSTER_FLAG, "l", "", "cluster id or ARN")
	execCmd.Flags().StringP(EXEC_TASK_FLAG, "t", "", "task id or ARN")
	execCmd.Flags().StringP(EXEC_CONTAINER_FLAG, "n", "", "container name")
	execCmd.Flags().StringP("command", "c", "/bin/sh", "command to run")
	execCmd.Flags().BoolP("interactive", "i", true, "toggles interactive mode")
}
