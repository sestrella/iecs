package exec

import (
	"context"
	"fmt"

	"github.com/aws/aws-sdk-go-v2/service/ecs/types"
	tea "github.com/charmbracelet/bubbletea"
	"github.com/charmbracelet/huh"
	"github.com/sestrella/iecs/client"
)

type Selector struct {
	form      *huh.Form
	cluster   *types.Cluster
	service   *types.Service
	task      *types.Task
	container *types.Container
}

func (model Selector) Init() tea.Cmd {
	return model.form.Init()
}

func (model Selector) Update(msg tea.Msg) (tea.Model, tea.Cmd) {
	switch msg := msg.(type) {
	case tea.KeyMsg:
		switch msg.String() {
		case "ctrl+c":
			return model, tea.Interrupt
		case "esc", "q":
			return model, tea.Quit
		}
	}

	var cmds []tea.Cmd
	form, cmd := model.form.Update(msg)
	if form, ok := form.(*huh.Form); ok {
		model.form = form
		cmds = append(cmds, cmd)
	}

	if model.form.State == huh.StateCompleted {
		cmds = append(cmds, tea.Quit)
	}

	return model, tea.Batch(cmds...)
}

func (model Selector) View() string {
	if model.form.State == huh.StateCompleted {
		return fmt.Sprintf(
			"Cluster: %s\nService: %s\nTask: %s\nContainer: %s",
			*model.cluster.ClusterName,
			*model.service.ServiceName,
			*model.task.TaskArn,
			*model.container.Name,
		)
	}

	return model.form.View()
}

func newSelector(ctx context.Context, client client.Client) (*Selector, error) {
	clusterArns, err := client.ListClusters(ctx)
	if err != nil {
		return nil, err
	}

	var clusterArn string
	var serviceArn string
	var taskArn string
	var containerName string
	form := huh.NewForm(
		huh.NewGroup(
			huh.NewSelect[string]().Title("Cluster").
				Options(huh.NewOptions(clusterArns...)...).
				Value(&clusterArn),
			huh.NewSelect[string]().Title("Service").
				OptionsFunc(func() []huh.Option[string] {
					serviceArns, err := client.ListServices(ctx, clusterArn)
					if err != nil {
						return nil
					}

					return huh.NewOptions(serviceArns...)
				}, &clusterArn).
				Value(&serviceArn),
		),
		huh.NewGroup(
			huh.NewSelect[string]().Title("Task").
				OptionsFunc(func() []huh.Option[string] {
					taskArns, err := client.ListTasks(ctx, clusterArn, serviceArn)
					if err != nil {
						return nil
					}

					return huh.NewOptions(taskArns...)
				}, &serviceArn).
				Value(&taskArn),
			huh.NewSelect[string]().Title("Container").
				OptionsFunc(func() []huh.Option[string] {
					tasks, err := client.DescribeTasks(ctx, clusterArn, []string{taskArn})
					if err != nil {
						return nil
					}

					task := tasks[0]

					var containerNames []string
					for _, container := range task.Containers {
						containerNames = append(containerNames, *container.Name)
					}

					return huh.NewOptions(containerNames...)
				}, &taskArn).
				Value(&containerName),
		),
	)

	if _, err := tea.NewProgram(form).Run(); err != nil {
		return nil, err
	}

	clusters, err := client.DescribeClusters(ctx, []string{clusterArn})
	if err != nil {
		return nil, err
	}

	cluster := clusters[0]

	services, err := client.DescribeServices(ctx, clusterArn, []string{serviceArn})
	if err != nil {
		return nil, err
	}

	service := services[0]

	tasks, err := client.DescribeTasks(ctx, clusterArn, []string{taskArn})
	if err != nil {
		return nil, err
	}

	task := tasks[0]

	var container types.Container
	for _, c := range task.Containers {
		if c.Name == &containerName {
			container = c
		}
	}

	return &Selector{
		form:      form,
		cluster:   &cluster,
		service:   &service,
		task:      &task,
		container: &container,
	}, nil
}
