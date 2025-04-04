package cmd

import (
	"context"
	"encoding/json"
	"fmt"
	"os"
	"os/exec"
	"os/signal"
	"strings"
	"syscall"

	"github.com/aws/aws-sdk-go-v2/config"
	"github.com/aws/aws-sdk-go-v2/service/ecs"
	"github.com/aws/aws-sdk-go-v2/service/ssm"
	"github.com/sestrella/iecs/selector"
	"github.com/spf13/cobra"
)

const (
	execClusterFlag     = "cluster"
	execServiceFlag     = "service"
	execTaskFlag        = "task"
	execContainerFlag   = "container"
	execCommandFlag     = "command"
	execInteractiveFlag = "interactive"
)

var execCmd = &cobra.Command{
	Use:   "exec",
	Short: "Run a remote command on a container",
	Example: `
  aws-vault exec <profile> -- iecs exec (recommended)
  env AWS_PROFILE=<profile> iecs exec
  `,
	Run: func(cmd *cobra.Command, args []string) {
		smpPath, err := exec.LookPath("session-manager-plugin")
		if err != nil {
			panic(err)
		}
		clusterId, err := cmd.Flags().GetString(execClusterFlag)
		if err != nil {
			panic(err)
		}
		serviceId, err := cmd.Flags().GetString(execServiceFlag)
		if err != nil {
			panic(err)
		}
		taskId, err := cmd.Flags().GetString(execTaskFlag)
		if err != nil {
			panic(err)
		}
		containerId, err := cmd.Flags().GetString(execContainerFlag)
		if err != nil {
			panic(err)
		}
		command, err := cmd.Flags().GetString(execCommandFlag)
		if err != nil {
			panic(err)
		}
		interactive, err := cmd.Flags().GetBool(execInteractiveFlag)
		if err != nil {
			panic(err)
		}
		cfg, err := config.LoadDefaultConfig(context.TODO())
		if err != nil {
			panic(err)
		}
		client := ecs.NewFromConfig(cfg)
		err = runExec(context.TODO(), smpPath, client, cfg.Region, clusterId, serviceId, taskId, containerId, command, interactive)
		if err != nil {
			panic(err)
		}
	},
	Aliases: []string{"ssh"},
}

func runExec(ctx context.Context, smpPath string, client *ecs.Client, region string, clusterId string, serviceId string, taskId string, containerId string, command string, interactive bool) error {
	defaultSelector := &selector.DefaultSelector{}
	cluster, err := selector.SelectCluster(ctx, client, defaultSelector, clusterId)
	if err != nil {
		return err
	}
	service, err := selector.SelectService(ctx, client, defaultSelector, *cluster.ClusterArn, serviceId)
	if err != nil {
		return err
	}
	task, err := selector.SelectTask(ctx, client, defaultSelector, *cluster.ClusterArn, *service.ServiceName, taskId)
	if err != nil {
		return err
	}
	container, err := selector.SelectContainer(task.Containers, containerId)
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
	cmd := exec.Command(smpPath,
		string(session),
		region,
		"StartSession",
		"",
		string(targetJSON),
		fmt.Sprintf("https://ssm.%s.amazonaws.com", region),
	)
	cmd.Stdin = os.Stdin
	cmd.Stdout = os.Stdout
	cmd.Stderr = os.Stderr

	err = cmd.Start()
	if err != nil {
		return err
	}

	// Reference: https://github.com/kubernetes/kubectl/blob/master/pkg/util/interrupt/interrupt.go
	go func() {
		sigs := make(chan os.Signal, 1)
		signal.Notify(sigs, syscall.SIGHUP, syscall.SIGINT, syscall.SIGQUIT, syscall.SIGTERM)
		sig := <-sigs
		err = cmd.Process.Signal(sig)
		if err != nil {
			panic(err)
		}
	}()

	return cmd.Wait()
}

func init() {
	rootCmd.AddCommand(execCmd)

	execCmd.Flags().StringP(execClusterFlag, "l", "", "cluster id or ARN")
	execCmd.Flags().StringP(execServiceFlag, "s", "", "service id or ARN")
	execCmd.Flags().StringP(execTaskFlag, "t", "", "task id or ARN")
	execCmd.Flags().StringP(execContainerFlag, "n", "", "container name")
	execCmd.Flags().StringP(execCommandFlag, "c", "/bin/bash", "command to run")
	execCmd.Flags().BoolP(execInteractiveFlag, "i", true, "toggles interactive mode")
}
