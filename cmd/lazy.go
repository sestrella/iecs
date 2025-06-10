package cmd

import (
	"context"
	"fmt"
	"log"

	"github.com/aws/aws-sdk-go-v2/config"
	"github.com/aws/aws-sdk-go-v2/service/ecs"
	"github.com/aws/aws-sdk-go-v2/service/ecs/types"
	"github.com/rivo/tview"
	"github.com/spf13/cobra"
)

type Lazy struct {
	ecs *ecs.Client

	app              *tview.Application
	clustersWidget   *tview.List
	servicesWidget   *tview.List
	tasksWidget      *tview.List
	containersWidget *tview.List
	logsWidget       *tview.TextView

	cluster    *types.Cluster
	clusters   []types.Cluster
	service    *types.Service
	services   []types.Service
	task       *types.Task
	tasks      []types.Task
	container  *types.Container
	containers []types.Container
}

func (lazy *Lazy) handleClusterSelection() {
	_, err := fmt.Fprintf(lazy.logsWidget, "Cluster %s selected\n", *lazy.cluster.ClusterName)
	if err != nil {
		log.Fatal(err)
	}

	listedServices, err := lazy.ecs.ListServices(
		context.TODO(),
		&ecs.ListServicesInput{
			Cluster: lazy.cluster.ClusterArn,
		},
	)
	if err != nil {
		log.Fatal(err)
	}

	describedServices, err := lazy.ecs.DescribeServices(
		context.TODO(),
		&ecs.DescribeServicesInput{
			Cluster:  lazy.cluster.ClusterArn,
			Services: listedServices.ServiceArns,
		},
	)
	if err != nil {
		log.Fatal(err)
	}

	lazy.services = describedServices.Services

	lazy.servicesWidget.Clear()
	for _, service := range lazy.services {
		lazy.servicesWidget.AddItem(
			*service.ServiceName,
			*service.ServiceArn,
			0,
			func() {},
		)
	}
	// lazy.app.SetFocus(lazy.servicesWidget)
}

func (lazy *Lazy) handleServiceSelection() {
	_, err := fmt.Fprintf(
		lazy.logsWidget,
		"Service %s selected\n",
		*lazy.service.ServiceName,
	)
	if err != nil {
		log.Fatal(err)
	}

	listedTasks, err := lazy.ecs.ListTasks(
		context.TODO(),
		&ecs.ListTasksInput{
			Cluster:     lazy.cluster.ClusterArn,
			ServiceName: lazy.service.ServiceName,
		},
	)
	if err != nil {
		log.Fatal(err)
	}

	describedTasks, err := lazy.ecs.DescribeTasks(
		context.TODO(),
		&ecs.DescribeTasksInput{
			Cluster: lazy.cluster.ClusterArn,
			Tasks:   listedTasks.TaskArns,
		},
	)
	if err != nil {
		log.Fatal(err)
	}

	lazy.tasks = describedTasks.Tasks

	lazy.tasksWidget.Clear()
	for _, task := range lazy.tasks {
		// TODO: Change task title
		lazy.tasksWidget.AddItem(*task.TaskArn, *task.TaskArn, 0, func() {})
	}
	// lazy.app.SetFocus(lazy.tasksWidget)
}

func (lazy *Lazy) handleTaskSelection() {
	_, err := fmt.Fprintf(
		lazy.logsWidget,
		"Task %s selected\n",
		*lazy.task.TaskArn,
	)
	if err != nil {
		log.Fatal(err)
	}

	lazy.containersWidget.Clear()
	for _, container := range lazy.containers {
		lazy.containersWidget.AddItem(
			*container.Name,
			*container.ContainerArn,
			0,
			func() {},
		)
	}
	// lazy.app.SetFocus(lazy.containersWidget)
}

// lazyCmd represents the lazy command
var lazyCmd = &cobra.Command{
	Use:   "lazy",
	Short: "A brief description of your command",
	RunE: func(cmd *cobra.Command, args []string) error {
		cfg, err := config.LoadDefaultConfig(context.TODO())
		if err != nil {
			return err
		}

		clusters := tview.NewList()
		clusters.SetTitle("Clusters")
		clusters.SetBorder(true)

		services := tview.NewList()
		services.SetTitle("Service")
		services.SetBorder(true)

		tasks := tview.NewList()
		tasks.SetTitle("Task")
		tasks.SetBorder(true)

		containers := tview.NewList()
		containers.SetTitle("Containers")
		containers.SetBorder(true)

		right := tview.NewFlex()
		right.SetDirection(tview.FlexRow)
		right.AddItem(clusters, 0, 1, false)
		right.AddItem(services, 0, 1, false)
		right.AddItem(tasks, 0, 1, false)
		right.AddItem(containers, 0, 1, false)

		main := tview.NewBox()
		main.SetTitle("Main")
		main.SetBorder(true)

		logs := tview.NewTextView()
		logs.SetTitle("Logs")
		logs.SetBorder(true)
		logs.SetDisabled(true)

		left := tview.NewFlex()
		left.SetDirection(tview.FlexRow)
		left.AddItem(main, 0, 4, false)
		left.AddItem(logs, 0, 1, false)

		root := tview.NewFlex()
		root.SetTitle("iecs")
		root.SetBorder(true)
		root.AddItem(right, 0, 1, false)
		root.AddItem(left, 0, 2, false)

		app := tview.NewApplication()

		lazy := &Lazy{
			ecs:              ecs.NewFromConfig(cfg),
			app:              app,
			clustersWidget:   clusters,
			servicesWidget:   services,
			tasksWidget:      tasks,
			containersWidget: containers,
			logsWidget:       logs,
		}

		lazy.clustersWidget.SetChangedFunc(
			func(index int, mainText, secondaryText string, shortcut rune) {
				lazy.cluster = &lazy.clusters[index]
				lazy.handleClusterSelection()
			},
		)

		lazy.servicesWidget.SetChangedFunc(
			func(index int, mainText, secondaryText string, shortcut rune) {
				lazy.service = &lazy.services[index]
				lazy.handleServiceSelection()
			},
		)

		lazy.tasksWidget.SetChangedFunc(
			func(index int, mainText, secondaryText string, shortcut rune) {
				lazy.task = &lazy.tasks[index]
				lazy.handleTaskSelection()
			},
		)

		lazy.containersWidget.SetChangedFunc(
			func(index int, mainText, secondaryText string, shortcut rune) {
				lazy.container = &lazy.containers[index]
			},
		)

		go func() {
			listedClusters, err := lazy.ecs.ListClusters(context.TODO(), &ecs.ListClustersInput{})
			if err != nil {
				log.Fatal(err)
			}

			describedClusters, err := lazy.ecs.DescribeClusters(
				context.TODO(),
				&ecs.DescribeClustersInput{
					Clusters: listedClusters.ClusterArns,
				},
			)
			if err != nil {
				log.Fatal(err)
			}

			lazy.clusters = describedClusters.Clusters

			app.QueueUpdateDraw(func() {
				clusters.Clear()
				for _, cluster := range lazy.clusters {
					clusters.AddItem(
						*cluster.ClusterName,
						*cluster.ClusterArn,
						0,
						func() {},
					)
				}
				app.SetFocus(clusters)
			})
		}()

		if err := app.SetRoot(root, true).Run(); err != nil {
			return err
		}

		return nil
	},
}

func init() {
	rootCmd.AddCommand(lazyCmd)
}
