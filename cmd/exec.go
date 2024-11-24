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

const (
	EXEC_COMMAND_FLAG     = "command"
	EXEC_INTERACTIVE_FLAG = "interactive"
)

var execCmd = &cobra.Command{
	Use:   "exec",
	Short: "Run a remote command on a container",
	Example: `
  aws-vault exec <profile> -- iecs exec (recommended)
  env AWS_PROFILE=<profile> iecs exec
  `,
	Run: func(cmd *cobra.Command, args []string) {
		clusterId, err := cmd.Flags().GetString(CLUSTER_FLAG)
		if err != nil {
			log.Fatal(err)
		}
		serviceId, err := cmd.Flags().GetString(SERVICE_FLAG)
		if err != nil {
			log.Fatal(err)
		}
		taskId, err := cmd.Flags().GetString(TASK_FLAG)
		if err != nil {
			log.Fatal(err)
		}
		containerId, err := cmd.Flags().GetString(CONTAINER_FLAG)
		if err != nil {
			log.Fatal(err)
		}
		command, err := cmd.Flags().GetString(EXEC_COMMAND_FLAG)
		if err != nil {
			log.Fatal(err)
		}
		interactive, err := cmd.Flags().GetBool(EXEC_INTERACTIVE_FLAG)
		if err != nil {
			log.Fatal(err)
		}
		cfg, err := config.LoadDefaultConfig(context.TODO())
		if err != nil {
			log.Fatal(err)
		}
		client := ecs.NewFromConfig(cfg)
		err = runExec(context.TODO(), client, cfg.Region, clusterId, serviceId, taskId, containerId, command, interactive)
		if err != nil {
			log.Fatal(err)
		}
	},
	Aliases: []string{"ssh"},
}

func runExec(ctx context.Context, client *ecs.Client, region string, clusterId string, serviceId string, taskId string, containerId string, command string, interactive bool) error {
	cluster, err := selectCluster(ctx, client, clusterId)
	if err != nil {
		return err
	}
	service, err := selectService(ctx, client, *cluster.ClusterArn, serviceId)
	if err != nil {
		return err
	}
	task, err := selectTask(ctx, client, *cluster.ClusterArn, *service.ServiceName, taskId)
	if err != nil {
		return err
	}
	container, err := selectContainer(task.Containers, containerId)
	if err != nil {
		return err
	}
	executeCommand, err := client.ExecuteCommand(ctx, &ecs.ExecuteCommandInput{
		Cluster:     cluster.ClusterArn,
		Task:        task.TaskArn,
		Container:   container.Name,
		Command:     &command,
		Interactive: interactive,
	})
	if err != nil {
		return err
	}
	session, err := json.Marshal(executeCommand.Session)
	if err != nil {
		return err
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

func selectContainer(containers []types.Container, containerId string) (*types.Container, error) {
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

	execCmd.Flags().StringP(EXEC_COMMAND_FLAG, "c", "/bin/bash", "command to run")
	execCmd.Flags().BoolP(EXEC_INTERACTIVE_FLAG, "i", true, "toggles interactive mode")
}
