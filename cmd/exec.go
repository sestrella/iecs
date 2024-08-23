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
	tea "github.com/charmbracelet/bubbletea"
	"github.com/charmbracelet/lipgloss"
	"github.com/spf13/cobra"
)

var docStyle = lipgloss.NewStyle().Margin(1, 2)

var clusterId string
var taskId string
var containerId string
var interactive bool
var command string

type model struct {
	list list.Model
}

func (m *model) Init() tea.Cmd {
	return nil
}

func (m *model) Update(msg tea.Msg) (tea.Model, tea.Cmd) {
	switch msg := msg.(type) {
	case tea.KeyMsg:
		switch msg.String() {
		case "enter":
			return m, tea.Quit
		case "q", "ctrl+c":
			// TODO: find a better way to kill bubbletea gracefully
			os.Exit(1)
			return m, tea.Quit
		}
	case tea.WindowSizeMsg:
		h, v := docStyle.GetFrameSize()
		m.list.SetSize(msg.Width-h, msg.Height-v)
	}

	var cmd tea.Cmd
	m.list, cmd = m.list.Update(msg)
	return m, cmd
}

func (m *model) View() string {
	return docStyle.Render(m.list.View())
}

type item struct {
	name      string
	arn       string
	runtimeId string
}

func (i item) Title() string       { return i.name }
func (i item) Description() string { return i.arn }
func (i item) FilterValue() string { return i.name }

// execCmd represents the exec command
var execCmd = &cobra.Command{
	Use:   "exec",
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
		selectedCluster, err := selectCluster(context.TODO(), client)
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

		var target = fmt.Sprintf("ecs:%s_%s_%s", selectedCluster.name, selectedTask.name, selectedContainer.runtimeId)
		targetJSON, err := json.Marshal(ssm.StartSessionInput{
			Target: &target,
		})
		if err != nil {
			log.Fatal(err)
		}

		fmt.Print("\nNon-interactive command:\n")
		fmt.Printf("\n\tlazy-ecs exec --cluster %s --task %s --container %s --command \"%s\" --interactive %t\n",
			selectedCluster.name,
			selectedTask.name,
			selectedContainer.name,
			command,
			interactive,
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

func selectCluster(ctx context.Context, client *ecs.Client) (*item, error) {
	if clusterId != "" {
		output, err := client.DescribeClusters(ctx, &ecs.DescribeClustersInput{
			Clusters: []string{clusterId},
		})
		if err != nil {
			return nil, err
		}

		if len(output.Clusters) == 0 {
			return nil, fmt.Errorf("Cluster '%s' not found", clusterId)
		}

		cluster := output.Clusters[0]
		return &item{
			name: *cluster.ClusterName,
			arn:  *cluster.ClusterArn,
		}, nil
	}

	output, err := client.ListClusters(ctx, &ecs.ListClustersInput{})
	if err != nil {
		return nil, err
	}
	if len(output.ClusterArns) == 0 {
		return nil, errors.New("No clusters found")
	}

	items := []list.Item{}
	for _, arn := range output.ClusterArns {
		index := strings.LastIndex(arn, "/")
		items = append(items, item{
			name: arn[index+1:],
			arn:  arn,
		})
	}
	return newSelector("Clusters", items)
}

func selectTask(ctx context.Context, client *ecs.Client, cluster item) (*item, error) {
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
			items = append(items, item{
				name:      *container.Name,
				arn:       *container.ContainerArn,
				runtimeId: *container.RuntimeId,
			})
		}
	}
	if len(items) == 0 {
		return nil, errors.New("No containers found")
	}

	return newSelector("Containers", items)
}

func newSelector(title string, items []list.Item) (*item, error) {
	list := list.New(items, list.NewDefaultDelegate(), 0, 0)
	list.Title = title

	model := model{list: list}
	if _, err := tea.NewProgram(&model).Run(); err != nil {
		return nil, err
	}

	selected := model.list.SelectedItem().(item)
	return &selected, nil
}

func init() {
	rootCmd.AddCommand(execCmd)

	// Here you will define your flags and configuration settings.

	// Cobra supports Persistent Flags which will work for this command
	// and all subcommands, e.g.:
	// execCmd.PersistentFlags().String("foo", "", "A help for foo")

	// Cobra supports local flags which will only run when this command
	// is called directly, e.g.:
	execCmd.Flags().StringVarP(&clusterId, "cluster", "c", "", "TODO")
	execCmd.Flags().StringVarP(&taskId, "task", "t", "", "TODO")
	execCmd.Flags().StringVarP(&containerId, "container", "n", "", "TODO")
	execCmd.Flags().StringVarP(&command, "command", "m", "/bin/sh", "TODO")
	execCmd.Flags().BoolVarP(&interactive, "interactive", "i", true, "TODO")
}
