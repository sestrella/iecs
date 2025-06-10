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
	ecs              *ecs.Client
	app              *tview.Application
	clustersWidget   *tview.List
	servicesWidget   *tview.List
	tasksWidget      *tview.List
	containersWidget *tview.List
	logsWidget       *tview.TextView
	clusters         []types.Cluster
	services         []types.Service
	tasks            []types.Task
}

func (lazy *Lazy) handleClusterSelection(cluster types.Cluster) {
	_, err := fmt.Fprintf(lazy.logsWidget, "Cluster %s selected\n", *cluster.ClusterName)
	if err != nil {
		log.Fatal(err)
	}

	listedServices, err := lazy.ecs.ListServices(
		context.TODO(),
		&ecs.ListServicesInput{
			Cluster: cluster.ClusterArn,
		},
	)
	if err != nil {
		log.Fatal(err)
	}

	describedServices, err := lazy.ecs.DescribeServices(
		context.TODO(),
		&ecs.DescribeServicesInput{
			Cluster:  cluster.ClusterArn,
			Services: listedServices.ServiceArns,
		},
	)
	if err != nil {
		log.Fatal(err)
	}

	lazy.services = describedServices.Services

	lazy.servicesWidget.SetChangedFunc(
		func(index int, mainText, secondaryText string, shortcut rune) {
			service := lazy.services[index]
			lazy.handleServiceSelection(cluster, service)
		},
	)

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

func (lazy *Lazy) handleServiceSelection(cluster types.Cluster, service types.Service) {
	_, err := fmt.Fprintf(
		lazy.logsWidget,
		"Service %s selected\n",
		*service.ServiceName,
	)
	if err != nil {
		log.Fatal(err)
	}

	listedTasks, err := lazy.ecs.ListTasks(
		context.TODO(),
		&ecs.ListTasksInput{
			Cluster:     cluster.ClusterArn,
			ServiceName: service.ServiceName,
		},
	)
	if err != nil {
		log.Fatal(err)
	}

	describedTasks, err := lazy.ecs.DescribeTasks(
		context.TODO(),
		&ecs.DescribeTasksInput{
			Cluster: cluster.ClusterArn,
			Tasks:   listedTasks.TaskArns,
		},
	)
	if err != nil {
		log.Fatal(err)
	}

	lazy.tasks = describedTasks.Tasks

	lazy.tasksWidget.SetChangedFunc(func(index int, mainText, secondaryText string, shortcut rune) {
		task := lazy.tasks[index]
		lazy.handleTaskSelection(task)
	})

	lazy.tasksWidget.Clear()
	for _, task := range lazy.tasks {
		// TODO: Change task title
		lazy.tasksWidget.AddItem(*task.TaskArn, *task.TaskArn, 0, func() {})
	}
	// lazy.app.SetFocus(lazy.tasksWidget)
}

func (lazy *Lazy) handleTaskSelection(task types.Task) {
	_, err := fmt.Fprintf(
		lazy.logsWidget,
		"Task %s selected\n",
		*task.TaskArn,
	)
	if err != nil {
		log.Fatal(err)
	}

	lazy.containersWidget.Clear()
	for _, container := range task.Containers {
		lazy.containersWidget.AddItem(
			*container.Name,
			*container.ContainerArn,
			0,
			func() {

			},
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

			clusters.SetChangedFunc(func(index int, mainText, secondaryText string, shortcut rune) {
				cluster := lazy.clusters[index]
				lazy.handleClusterSelection(cluster)
			})

			app.QueueUpdateDraw(func() {
				clusters.Clear()
				for _, cluster := range lazy.clusters {
					currentCluster := cluster // Create a local copy to avoid closure issues
					clusters.AddItem(
						*currentCluster.ClusterName,
						*currentCluster.ClusterArn,
						0,
						func() {},
					)
					app.SetFocus(clusters)
				}
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
