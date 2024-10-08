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
	"github.com/aws/aws-sdk-go-v2/service/ssm"
	"github.com/charmbracelet/bubbles/list"
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

		task, err := selectTask(context.TODO(), client, *cluster, taskId)
		if err != nil {
			log.Fatal(err)
		}

		container, err := selectContainer(context.TODO(), client, *cluster, *task, containerId)
		if err != nil {
			log.Fatal(err)
		}

		output, err := client.ExecuteCommand(context.TODO(), &ecs.ExecuteCommandInput{
			Cluster:     &cluster.arn,
			Task:        &task.arn,
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

		var target = fmt.Sprintf("ecs:%s_%s_%s", cluster.name, task.name, container.runtimeId)
		targetJSON, err := json.Marshal(ssm.StartSessionInput{
			Target: &target,
		})
		if err != nil {
			log.Fatal(err)
		}

		fmt.Print("\nNon-interactive command:\n")
		fmt.Printf("\n\tlazy-ecs ssh --cluster %s --task %s --container %s --command \"%s\" --interactive %t\n",
			cluster.name,
			task.name,
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

func selectTask(ctx context.Context, client *ecs.Client, cluster item, taskId string) (*item, error) {
	if taskId != "" {
		output, err := client.DescribeTasks(ctx, &ecs.DescribeTasksInput{
			Cluster: &cluster.arn,
			Tasks:   []string{taskId},
		})
		if err != nil {
			return nil, err
		}

		if len(output.Tasks) == 0 {
			return nil, fmt.Errorf("Task '%s' not found", taskId)
		}

		task := output.Tasks[0]
		slices := strings.Split(*task.TaskArn, "/")
		return &item{
			name: fmt.Sprintf("%s/%s", slices[1], slices[2]),
			arn:  *task.TaskArn,
		}, nil
	}

	output, err := client.ListTasks(ctx, &ecs.ListTasksInput{
		Cluster: &cluster.arn,
	})
	if err != nil {
		return nil, err
	}
	if len(output.TaskArns) == 0 {
		return nil, errors.New("No tasks found")
	}

	items := []list.Item{}
	for _, arn := range output.TaskArns {
		slices := strings.Split(arn, "/")
		items = append(items, item{
			name: fmt.Sprintf("%s/%s", slices[1], slices[2]),
			arn:  arn,
		})
	}
	return newSelector("Tasks", items)
}

func selectContainer(ctx context.Context, client *ecs.Client, cluster item, task item, containerId string) (*item, error) {
	output, err := client.DescribeTasks(ctx, &ecs.DescribeTasksInput{
		Cluster: &cluster.arn,
		Tasks:   []string{task.arn},
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
