package cmd

import (
	"context"
	"fmt"
	"log"
	"time"

	"github.com/aws/aws-sdk-go-v2/config"
	cwl "github.com/aws/aws-sdk-go-v2/service/cloudwatchlogs"
	"github.com/aws/aws-sdk-go-v2/service/cloudwatchlogs/types"
	"github.com/aws/aws-sdk-go-v2/service/ecs"
	ecsTypes "github.com/aws/aws-sdk-go-v2/service/ecs/types"
	"github.com/gdamore/tcell/v2"
	"github.com/rivo/tview"
	"github.com/spf13/cobra"
)

type Lazy struct {
	ecs *ecs.Client
	cwl *cwl.Client

	app              *tview.Application
	clustersWidget   *tview.List
	servicesWidget   *tview.List
	tasksWidget      *tview.List
	containersWidget *tview.List
	logsWidget       *tview.TextView
	main             *tview.Pages

	cluster   *ecsTypes.Cluster
	clusters  []ecsTypes.Cluster
	service   *ecsTypes.Service
	task      *ecsTypes.Task
	container *ecsTypes.Container

	servicesByTask map[string][]ecsTypes.Service
	tasksByService map[string][]ecsTypes.Task
}

func (lazy *Lazy) handleClusterSelection() {
	lazy.log("Cluster %s selected\n", *lazy.cluster.ClusterName)

	services, ok := lazy.servicesByTask[*lazy.cluster.ClusterArn]

	if !ok {
		lazy.log("Fetching services for cluster %s", *lazy.cluster.ClusterName)

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

		services = describedServices.Services
		lazy.servicesByTask[*lazy.cluster.ClusterArn] = services
	}

	lazy.servicesWidget.Clear()
	for _, service := range services {
		lazy.servicesWidget.AddItem(
			*service.ServiceName,
			*service.ServiceArn,
			0,
			func() {},
		)
	}
}

func (lazy *Lazy) handleServiceSelection() {
	lazy.log("Service %s selected\n", *lazy.service.ServiceName)

	tasks, ok := lazy.tasksByService[*lazy.service.ServiceArn]

	if !ok {
		lazy.log("Fetching tasks for service %s", *lazy.service.ServiceArn)

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

		tasks = describedTasks.Tasks
		lazy.tasksByService[*lazy.service.ServiceArn] = tasks
	}

	lazy.tasksWidget.Clear()
	for _, task := range tasks {
		lazy.tasksWidget.AddItem(*task.TaskArn, *task.TaskArn, 0, func() {})
	}
}

func (lazy *Lazy) handleTaskSelection() {
	lazy.log("Task %s selected\n", *lazy.task.TaskArn)

	lazy.containersWidget.Clear()
	for _, container := range lazy.task.Containers {
		lazy.containersWidget.AddItem(
			*container.Name,
			*container.ContainerArn,
			0,
			func() {},
		)
	}
}

func (lazy *Lazy) log(format string, a ...any) {
	_, err := fmt.Fprintf(lazy.logsWidget, format, a...)
	if err != nil {
		log.Fatal(err)
	}

	lazy.logsWidget.ScrollToEnd()
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

		clustersWidget := tview.NewList()
		clustersWidget.SetTitle("Clusters (1)")
		clustersWidget.SetBorder(true)

		servicesWidget := tview.NewList()
		servicesWidget.SetTitle("Services (2)")
		servicesWidget.SetBorder(true)

		tasksWidget := tview.NewList()
		tasksWidget.SetTitle("Tasks (3)")
		tasksWidget.SetBorder(true)

		containersWidget := tview.NewList()
		containersWidget.SetTitle("Containers (4)")
		containersWidget.SetBorder(true)

		right := tview.NewFlex()
		right.SetDirection(tview.FlexRow)
		right.AddItem(clustersWidget, 0, 1, false)
		right.AddItem(servicesWidget, 0, 1, false)
		right.AddItem(tasksWidget, 0, 1, false)
		right.AddItem(containersWidget, 0, 1, false)

		main := tview.NewPages()
		main.SetTitle("Main (5)")
		main.SetBorder(true)

		logs := tview.NewTextView()
		logs.SetTitle("Logs (6)")
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
		app.SetInputCapture(func(event *tcell.EventKey) *tcell.EventKey {
			if event.Rune() == '1' {
				app.SetFocus(clustersWidget)
				return nil
			}
			if event.Rune() == '2' {
				app.SetFocus(servicesWidget)
				return nil
			}
			if event.Rune() == '3' {
				app.SetFocus(tasksWidget)
				return nil
			}
			if event.Rune() == '4' {
				app.SetFocus(containersWidget)
				return nil
			}
			if event.Rune() == '5' {
				app.SetFocus(main)
				return nil
			}
			if event.Rune() == '6' {
				app.SetFocus(logs)
				return nil
			}
			return event
		})

		lazy := &Lazy{
			ecs:              ecs.NewFromConfig(cfg),
			cwl:              cwl.NewFromConfig(cfg),
			app:              app,
			clustersWidget:   clustersWidget,
			servicesWidget:   servicesWidget,
			tasksWidget:      tasksWidget,
			containersWidget: containersWidget,
			logsWidget:       logs,
			main:             main,
			servicesByTask:   make(map[string][]ecsTypes.Service),
			tasksByService:   make(map[string][]ecsTypes.Task),
		}

		lazy.clustersWidget.SetChangedFunc(
			func(index int, mainText, secondaryText string, shortcut rune) {
				lazy.cluster = &lazy.clusters[index]
				lazy.handleClusterSelection()
			},
		)

		lazy.servicesWidget.SetChangedFunc(
			func(index int, mainText, secondaryText string, shortcut rune) {
				lazy.service = &lazy.servicesByTask[*lazy.cluster.ClusterArn][index]
				lazy.handleServiceSelection()
			},
		)
		lazy.servicesWidget.SetInputCapture(func(event *tcell.EventKey) *tcell.EventKey {
			if event.Rune() == 'l' {
				lazy.log("Tailing logs for service %s\n", *lazy.service.ServiceName)
				go func() {
					describedTaskDefinition, err := lazy.ecs.DescribeTaskDefinition(
						context.TODO(),
						&ecs.DescribeTaskDefinitionInput{
							TaskDefinition: lazy.service.TaskDefinition,
						},
					)
					if err != nil {
						panic(err)
					}

					var logGroupArns []string
					for _, containerDefinition := range describedTaskDefinition.TaskDefinition.ContainerDefinitions {
						logOptions := containerDefinition.LogConfiguration.Options
						logGroupName := logOptions["awslogs-group"]
						describedLogGroups, err := lazy.cwl.DescribeLogGroups(
							context.TODO(),
							&cwl.DescribeLogGroupsInput{
								LogGroupNamePrefix: &logGroupName,
							},
						)
						if err != nil {
							panic(err)
						}
						logGroups := describedLogGroups.LogGroups
						if len(logGroups) == 1 {
							logGroupArns = append(logGroupArns, *logGroups[0].LogGroupArn)
						}
					}
					startedLiveTail, err := lazy.cwl.StartLiveTail(
						context.TODO(),
						&cwl.StartLiveTailInput{
							LogGroupIdentifiers: logGroupArns,
						},
					)
					if err != nil {
						panic(err)
					}

					stream := startedLiveTail.GetStream()
					defer func() {
						err := stream.Close()
						if err != nil {
							panic(err)
						}
					}()

					eventsChan := stream.Events()

					// TODO: check if the page already exists
					serviceLogs := tview.NewTextView()
					main.AddAndSwitchToPage(*lazy.service.ServiceArn, serviceLogs, true)

					for {
						event := <-eventsChan
						switch e := event.(type) {
						case *types.StartLiveTailResponseStreamMemberSessionUpdate:
							for _, logEvent := range e.Value.SessionResults {
								timestamp := time.UnixMilli(*logEvent.Timestamp)
								app.QueueUpdateDraw(func() {
									_, err := fmt.Fprintf(serviceLogs, "%v %s %s\n", timestamp, *logEvent.Message, *logEvent.LogStreamName)
									if err != nil {
										panic(err)
									}
									serviceLogs.ScrollToEnd()
								})
							}
						}
					}
				}()
				return nil
			}
			return event
		})

		lazy.tasksWidget.SetChangedFunc(
			func(index int, mainText, secondaryText string, shortcut rune) {
				lazy.task = &lazy.tasksByService[*lazy.service.ServiceArn][index]

				// if lazy.app.GetFocus() == lazy.tasksWidget {
				// 	content, err := json.MarshalIndent(lazy.task, "", "  ")
				// 	if err != nil {
				// 		// TODO: Do something
				// 		log.Fatal(err)
				// 	}
				// 	lazy.main.SetText(string(content))
				// }

				lazy.handleTaskSelection()
			},
		)

		lazy.containersWidget.SetChangedFunc(
			func(index int, mainText, secondaryText string, shortcut rune) {
				lazy.container = &lazy.task.Containers[index]
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
				clustersWidget.Clear()
				for _, cluster := range lazy.clusters {
					clustersWidget.AddItem(
						*cluster.ClusterName,
						*cluster.ClusterArn,
						0,
						func() {},
					)
				}
				app.SetFocus(clustersWidget)
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
