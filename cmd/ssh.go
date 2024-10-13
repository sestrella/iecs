package cmd

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"log"
	"os"
	"os/exec"
	"strings"

	"github.com/aws/aws-sdk-go-v2/config"
	"github.com/aws/aws-sdk-go-v2/service/ecs"
	"github.com/aws/aws-sdk-go-v2/service/ecs/types"
	"github.com/aws/aws-sdk-go-v2/service/ssm"
	"github.com/charmbracelet/bubbles/list"
	"github.com/pterm/pterm"
	"github.com/spf13/cobra"
)

// sshCmd represents the exec command
var sshCmd = &cobra.Command{
	Use:   "ssh",
	Short: "A brief description of your command",
	Long:  "TODO",
	Run: func(cmd *cobra.Command, args []string) {
		clusterId, _ := cmd.Flags().GetString("cluster")
		taskId, _ := cmd.Flags().GetString("task")
		containerId, _ := cmd.Flags().GetString("container")
		command, _ := cmd.Flags().GetString("command")
		interactive, _ := cmd.Flags().GetBool("interactive")

		cfg, err := config.LoadDefaultConfig(context.TODO())
		if err != nil {
			log.Fatal(err)
		}

		client := ecs.NewFromConfig(cfg)
		cluster, err := selectCluster(context.TODO(), client, clusterId)
		if err != nil {
			log.Fatal(err)
		}

		task, err := selectTask(context.TODO(), client, *cluster.ClusterName, taskId)
		if err != nil {
			log.Fatal(err)
		}

		container, err := selectContainer(context.TODO(), client, *cluster.ClusterName, *task.TaskArn, containerId)
		if err != nil {
			log.Fatal(err)
		}

		output, err := client.ExecuteCommand(context.TODO(), &ecs.ExecuteCommandInput{
			Cluster:     cluster.ClusterArn,
			Task:        task.TaskArn,
			Container:   &container.name,
			Command:     &command,
			Interactive: interactive,
		})
		if err != nil {
			log.Fatal(err)
		}

		session, err := json.Marshal(output.Session)
		if err != nil {
			log.Fatal(err)
		}

		foo := strings.Split(*task.TaskArn, "/")
		var target = fmt.Sprintf("ecs:%s_%s_%s", *cluster.ClusterName, foo[0], container.runtimeId)
		targetJSON, err := json.Marshal(ssm.StartSessionInput{
			Target: &target,
		})
		if err != nil {
			log.Fatal(err)
		}

		fmt.Print("\nNon-interactive command:\n")
		fmt.Printf("\n\tlazy-ecs ssh --cluster %s --task %s --container %s --command \"%s\" --interactive %t\n",
			*cluster.ClusterName,
			*task.TaskArn,
			container.name,
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

		err = smp.Run()
		if err != nil {
			log.Fatal(err)
		}
	},
}

func selectTask(ctx context.Context, client *ecs.Client, clusterId string, taskId string) (*types.Task, error) {
	if taskId != "" {
		return describeTask(ctx, client, clusterId, taskId)
	}

	output, _ := client.ListTasks(ctx, &ecs.ListTasksInput{
		Cluster: &clusterId,
	})
	// if len(output.TaskArns) == 0 {
	// 	return nil, errors.New("No tasks found")
	// }
	//
	taskArn, _ := pterm.DefaultInteractiveSelect.WithOptions(output.TaskArns).Show()
	return describeTask(ctx, client, clusterId, taskArn)
}

func describeTask(ctx context.Context, client *ecs.Client, clusterId string, taskId string) (*types.Task, error) {
	output, _ := client.DescribeTasks(ctx, &ecs.DescribeTasksInput{
		Cluster: &clusterId,
		Tasks:   []string{taskId},
	})
	task := output.Tasks[0]
	return &task, nil
}

func selectContainer(ctx context.Context, client *ecs.Client, cluster string, task string, containerId string) (*item, error) {
	output, err := client.DescribeTasks(ctx, &ecs.DescribeTasksInput{
		Cluster: &cluster,
		Tasks:   []string{task},
	})
	if err != nil {
		return nil, err
	}
	if len(output.Tasks) == 0 {
		return nil, errors.New("No tasks found")
	}

	items := []list.Item{}
	for _, task := range output.Tasks {
		for _, container := range task.Containers {
			var containerItem = item{
				name:      *container.Name,
				arn:       *container.ContainerArn,
				runtimeId: *container.RuntimeId,
			}
			if containerId != "" && *container.Name == containerId {
				return &containerItem, nil
			}
			items = append(items, containerItem)
		}
	}
	if containerId != "" {
		return nil, fmt.Errorf("Container '%s' not found", containerId)
	}
	if len(items) == 0 {
		return nil, errors.New("No containers found")
	}

	return newSelector("Containers", items)
}

func init() {
	rootCmd.AddCommand(sshCmd)

	sshCmd.Flags().StringP("cluster", "c", "", "TODO")
	sshCmd.Flags().StringP("task", "t", "", "TODO")
	sshCmd.Flags().StringP("container", "n", "", "TODO")
	sshCmd.Flags().StringP("command", "m", "/bin/sh", "TODO")
	sshCmd.Flags().BoolP("interactive", "i", true, "TODO")
}
