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

var sshClusterId string
var sshTaskId string
var sshContainerId string
var sshInteractive bool
var sshCommand string

// sshCmd represents the exec command
var sshCmd = &cobra.Command{
	Use:   "ssh",
	Short: "A brief description of your command",
	Long: `A longer description that spans multiple lines and likely contains examples
and usage of using your command. For example:

Cobra is a CLI library for Go that empowers applications.
This application is a tool to generate the needed files
to quickly create a Cobra application.`,
	Run: func(cmd *cobra.Command, args []string) {
		cfg, err := config.LoadDefaultConfig(context.TODO())
		if err != nil {
			log.Fatal(err)
		}

		client := ecs.NewFromConfig(cfg)
		selectedCluster, err := selectCluster(context.TODO(), client, sshClusterId)
		if err != nil {
			log.Fatal(err)
		}

		selectedTask, err := selectTask(context.TODO(), client, *selectedCluster)
		if err != nil {
			log.Fatal(err)
		}

		selectedContainer, err := selectContainer(client, *selectedCluster, *selectedTask)
		if err != nil {
			log.Fatal(err)
		}

		output, err := client.ExecuteCommand(context.TODO(), &ecs.ExecuteCommandInput{
			Cluster:     &selectedCluster.arn,
			Task:        &selectedTask.arn,
			Container:   &selectedContainer.name,
			Command:     &sshCommand,
			Interactive: sshInteractive,
		})
		if err != nil {
			log.Fatal(err)
		}

		session, err := json.Marshal(output.Session)
		if err != nil {
			log.Fatal(err)
		}

		var target = fmt.Sprintf("ecs:%s_%s_%s", selectedCluster.name, selectedTask.name, selectedContainer.runtimeId)
		targetJSON, err := json.Marshal(ssm.StartSessionInput{
			Target: &target,
		})
		if err != nil {
			log.Fatal(err)
		}

		fmt.Print("\nNon-interactive command:\n")
		fmt.Printf("\n\tlazy-ecs ssh --cluster %s --task %s --container %s --command \"%s\" --interactive %t\n",
			selectedCluster.name,
			selectedTask.name,
			selectedContainer.name,
			sshCommand,
			sshInteractive,
		)

		// https://github.com/aws/aws-cli/blob/develop/awscli/customizations/ecs/executecommand.py
		smp := exec.Command("session-manager-plugin",
			string(session),
			cfg.Region,
			"StartSession",
			"",
			string(targetJSON),
			fmt.Sprintf("https://ssm.%s.amazonaws.com", cfg.Region),
		)
		smp.Stdin = os.Stdin
		smp.Stdout = os.Stdout
		smp.Stderr = os.Stderr

		err = smp.Run()
		if err != nil {
			log.Fatal(err)
		}
	},
}

func selectTask(ctx context.Context, client *ecs.Client, cluster item) (*item, error) {
	if sshTaskId != "" {
		output, err := client.DescribeTasks(ctx, &ecs.DescribeTasksInput{
			Cluster: &cluster.arn,
			Tasks:   []string{sshTaskId},
		})
		if err != nil {
			return nil, err
		}

		if len(output.Tasks) == 0 {
			return nil, fmt.Errorf("Task '%s' not found", sshTaskId)
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

func selectContainer(client *ecs.Client, cluster item, task item) (*item, error) {
	output, err := client.DescribeTasks(context.TODO(), &ecs.DescribeTasksInput{
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
			if sshContainerId != "" && *container.Name == sshContainerId {
				return &containerItem, nil
			}
			items = append(items, containerItem)
		}
	}
	if sshContainerId != "" {
		return nil, fmt.Errorf("Container '%s' not found", sshContainerId)
	}
	if len(items) == 0 {
		return nil, errors.New("No containers found")
	}

	return newSelector("Containers", items)
}

func init() {
	rootCmd.AddCommand(sshCmd)

	// Here you will define your flags and configuration settings.

	// Cobra supports Persistent Flags which will work for this command
	// and all subcommands, e.g.:
	// execCmd.PersistentFlags().String("foo", "", "A help for foo")

	// Cobra supports local flags which will only run when this command
	// is called directly, e.g.:
	sshCmd.Flags().StringVarP(&sshClusterId, "cluster", "c", "", "TODO")
	sshCmd.Flags().StringVarP(&sshTaskId, "task", "t", "", "TODO")
	sshCmd.Flags().StringVarP(&sshContainerId, "container", "n", "", "TODO")
	sshCmd.Flags().StringVarP(&sshCommand, "command", "m", "/bin/sh", "TODO")
	sshCmd.Flags().BoolVarP(&sshInteractive, "interactive", "i", true, "TODO")
}
