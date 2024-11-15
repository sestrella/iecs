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

const SSH_CLUSTER_FLAG = "cluster"
const SSH_SERVICE_FLAG = "service"
const SSH_TASK_FLAG = "task"
const SSH_CONTAINER_FLAG = "container"
const SSH_COMMAND_FLAG = "command"
const SSH_INTERACTIVE_FLAG = "interactive"

var sshCmd = &cobra.Command{
	Use:   "ssh",
	Short: "Run a remote command on a container",
	Example: `
  aws-vault exec <profile> -- iecs ssh (recommended)
  env AWS_PROFILE=<profile> iecs ssh
  `,
	Run: func(cmd *cobra.Command, args []string) {
		clusterId, err := cmd.Flags().GetString(SSH_CLUSTER_FLAG)
		if err != nil {
			log.Fatal(err)
		}
		taskId, err := cmd.Flags().GetString(SSH_TASK_FLAG)
		if err != nil {
			log.Fatal(err)
		}
		containerId, err := cmd.Flags().GetString(SSH_CONTAINER_FLAG)
		if err != nil {
			log.Fatal(err)
		}
		command, err := cmd.Flags().GetString(SSH_COMMAND_FLAG)
		if err != nil {
			log.Fatal(err)
		}
		interactive, err := cmd.Flags().GetBool(SSH_INTERACTIVE_FLAG)
		if err != nil {
			log.Fatal(err)
		}
		cfg, err := config.LoadDefaultConfig(context.TODO())
		if err != nil {
			log.Fatal(err)
		}
		client := ecs.NewFromConfig(cfg)
		err = runSsh(context.TODO(), client, cfg.Region, clusterId, taskId, containerId, command, interactive)
		if err != nil {
			log.Fatal(err)
		}
	},
	Aliases: []string{"exec"},
}

func runSsh(ctx context.Context, client *ecs.Client, region string, clusterId string, taskId string, containerId string, command string, interactive bool) error {
	cluster, err := describeCluster(ctx, client, clusterId)
	if err != nil {
		return err
	}
	service, err := describeService(ctx, client, *cluster.ClusterArn, "")
	if err != nil {
		return err
	}
	task, err := describeTask(ctx, client, *cluster.ClusterArn, *service.ServiceName, taskId)
	if err != nil {
		return err
	}
	container, err := describeContainer(task.Containers, containerId)
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
	rootCmd.AddCommand(sshCmd)

	sshCmd.Flags().StringP(SSH_CLUSTER_FLAG, "l", "", "cluster id or ARN")
	sshCmd.Flags().StringP(SSH_SERVICE_FLAG, "s", "", "service id or ARN")
	sshCmd.Flags().StringP(SSH_TASK_FLAG, "t", "", "task id or ARN")
	sshCmd.Flags().StringP(SSH_CONTAINER_FLAG, "n", "", "container name")
	sshCmd.Flags().StringP(SSH_COMMAND_FLAG, "c", "/bin/bash", "command to run")
	sshCmd.Flags().BoolP(SSH_INTERACTIVE_FLAG, "i", true, "toggles interactive mode")
}
