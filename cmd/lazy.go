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

// lazyCmd represents the lazy command
var lazyCmd = &cobra.Command{
	Use:   "lazy",
	Short: "A brief description of your command",
	RunE: func(cmd *cobra.Command, args []string) error {
		cfg, err := config.LoadDefaultConfig(context.TODO())
		if err != nil {
			return err
		}

		ecsService := ecs.NewFromConfig(cfg)

		clusters := tview.NewList()
		clusters.SetTitle("Clusters")
		clusters.SetBorder(true)

		services := tview.NewList()
		services.SetTitle("Service")
		services.SetBorder(true)
		services.AddItem("", "", 0, func() {})

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

		go func() {
			listedClusters, err := ecsService.ListClusters(context.TODO(), &ecs.ListClustersInput{})
			if err != nil {
				log.Fatal(err)
			}

			describedClusters, err := ecsService.DescribeClusters(
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
					clusters.AddItem(*cluster.ClusterName, *cluster.ClusterArn, 0, func() {
						handleClusterSelection(
							app,
							logs,
							ecsService,
							cluster,
							services,
							tasks,
							containers,
						)
					})
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

func handleClusterSelection(
	app *tview.Application,
	logs *tview.TextView,
	ecsService *ecs.Client,
	cluster types.Cluster,
	services *tview.List,
	tasks *tview.List,
	containers *tview.List,
) {
	_, err := fmt.Fprintf(logs, "Cluster %s selected\n", *cluster.ClusterName)
	if err != nil {
		log.Fatal(err)
	}

	listedServices, err := ecsService.ListServices(
		context.TODO(),
		&ecs.ListServicesInput{
			Cluster: cluster.ClusterArn,
		},
	)
	if err != nil {
		log.Fatal(err)
	}

	describedServices, err := ecsService.DescribeServices(
		context.TODO(),
		&ecs.DescribeServicesInput{
			Cluster:  cluster.ClusterArn,
			Services: listedServices.ServiceArns,
		},
	)
	if err != nil {
		log.Fatal(err)
	}

	services.Clear()
	for _, service := range describedServices.Services {
		services.AddItem(*service.ServiceName, *service.ServiceArn, 0, func() {
			_, err := fmt.Fprintf(
				logs,
				"Service %s selected\n",
				*service.ServiceName,
			)
			if err != nil {
				log.Fatal(err)
			}

			listedTasks, err := ecsService.ListTasks(
				context.TODO(),
				&ecs.ListTasksInput{
					Cluster:     cluster.ClusterArn,
					ServiceName: service.ServiceName,
				},
			)
			if err != nil {
				log.Fatal(err)
			}

			describedTasks, err := ecsService.DescribeTasks(
				context.TODO(),
				&ecs.DescribeTasksInput{
					Cluster: cluster.ClusterArn,
					Tasks:   listedTasks.TaskArns,
				},
			)
			if err != nil {
				log.Fatal(err)
			}

			tasks.Clear()
			for _, task := range describedTasks.Tasks {
				// TODO: Change task title
				tasks.AddItem(*task.TaskArn, *task.TaskArn, 0, func() {
					_, err := fmt.Fprintf(
						logs,
						"Task %s selected\n",
						*task.TaskArn,
					)
					if err != nil {
						log.Fatal(err)
					}

					containers.Clear()
					for _, container := range task.Containers {
						containers.AddItem(
							*container.Name,
							*container.ContainerArn,
							0,
							func() {

							},
						)
					}
					app.SetFocus(containers)
				})
			}
			app.SetFocus(tasks)
		})
	}
	app.SetFocus(services)
}
