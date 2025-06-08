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
	ecs        *ecs.Client
	app        *tview.Application
	clusters   *tview.List
	services   *tview.List
	tasks      *tview.List
	containers *tview.List
	logs       *tview.TextView
}

func (lazy *Lazy) handleClusterSelection(cluster types.Cluster) {
	_, err := fmt.Fprintf(lazy.logs, "Cluster %s selected\n", *cluster.ClusterName)
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

	lazy.services.Clear()
	for _, service := range describedServices.Services {
		currentService := service // Create a local copy to avoid closure issues
		lazy.services.AddItem(*currentService.ServiceName, *currentService.ServiceArn, 0, func() {
			lazy.handleServiceSelection(cluster, currentService)
		})
	}
	lazy.app.SetFocus(lazy.services)
}

func (lazy *Lazy) handleServiceSelection(cluster types.Cluster, service types.Service) {
	_, err := fmt.Fprintf(
		lazy.logs,
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

	lazy.tasks.Clear()
	for _, task := range describedTasks.Tasks {
		// TODO: Change task title
		currentTask := task // Create a local copy to avoid closure issues
		lazy.tasks.AddItem(*currentTask.TaskArn, *currentTask.TaskArn, 0, func() {
			lazy.handleTaskSelection(currentTask)
		})
	}
	lazy.app.SetFocus(lazy.tasks)
}

func (lazy *Lazy) handleTaskSelection(task types.Task) {
	_, err := fmt.Fprintf(
		lazy.logs,
		"Task %s selected\n",
		*task.TaskArn,
	)
	if err != nil {
		log.Fatal(err)
	}

	lazy.containers.Clear()
	for _, container := range task.Containers {
		lazy.containers.AddItem(
			*container.Name,
			*container.ContainerArn,
			0,
			func() {

			},
		)
	}
	lazy.app.SetFocus(lazy.containers)
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
			ecs:        ecs.NewFromConfig(cfg),
			app:        app,
			clusters:   clusters,
			services:   services,
			tasks:      tasks,
			containers: containers,
			logs:       logs,
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

			app.QueueUpdateDraw(func() {
				clusters.Clear()
				for _, cluster := range describedClusters.Clusters {
					currentCluster := cluster // Create a local copy to avoid closure issues
					clusters.AddItem(
						*currentCluster.ClusterName,
						*currentCluster.ClusterArn,
						0,
						func() {
							lazy.handleClusterSelection(currentCluster)
						},
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
