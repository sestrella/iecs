package cmd

import (
	"context"
	"encoding/json"
	"fmt"
	"log"
	"strings"
	"time"

	"github.com/aws/aws-sdk-go-v2/config"
	cwl "github.com/aws/aws-sdk-go-v2/service/cloudwatchlogs"
	cwlTypes "github.com/aws/aws-sdk-go-v2/service/cloudwatchlogs/types"
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

	cluster          *ecsTypes.Cluster
	clusters         []ecsTypes.Cluster
	service          *ecsTypes.Service
	task             *ecsTypes.Task
	container        *ecsTypes.Container
	expandMainWidget bool

	servicesByTask map[string][]ecsTypes.Service
	tasksByService map[string][]ecsTypes.Task
}

type Handlers struct {
	start  func()
	update func(logEvent cwlTypes.LiveTailSessionLogEvent)
}

func (lazy *Lazy) handleClusterSelection(ctx context.Context, cluster ecsTypes.Cluster) error {
	lazy.log("Cluster %s selected\n", *cluster.ClusterName)
	services, ok := lazy.servicesByTask[*cluster.ClusterArn]
	if !ok {
		lazy.log("Fetching services for cluster %s", *cluster.ClusterName)
		listedServices, err := lazy.ecs.ListServices(
			ctx,
			&ecs.ListServicesInput{
				Cluster: cluster.ClusterArn,
			},
		)
		if err != nil {
			return err
		}
		describedServices, err := lazy.ecs.DescribeServices(
			ctx,
			&ecs.DescribeServicesInput{
				Cluster:  cluster.ClusterArn,
				Services: listedServices.ServiceArns,
			},
		)
		if err != nil {
			return err
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
	return nil
}

func (lazy *Lazy) handleServiceSelection(ctx context.Context, service ecsTypes.Service) error {
	lazy.log("Service %s selected\n", *service.ServiceName)
	tasks, ok := lazy.tasksByService[*service.ServiceArn]
	if !ok {
		lazy.log("Fetching tasks for service %s", *service.ServiceArn)
		listedTasks, err := lazy.ecs.ListTasks(
			ctx,
			&ecs.ListTasksInput{
				Cluster:     lazy.cluster.ClusterArn,
				ServiceName: service.ServiceName,
			},
		)
		if err != nil {
			return err
		}
		describedTasks, err := lazy.ecs.DescribeTasks(
			ctx,
			&ecs.DescribeTasksInput{
				Cluster: lazy.cluster.ClusterArn,
				Tasks:   listedTasks.TaskArns,
			},
		)
		if err != nil {
			return err
		}
		tasks = describedTasks.Tasks
		lazy.tasksByService[*service.ServiceArn] = tasks
	}
	lazy.tasksWidget.Clear()
	for _, task := range tasks {
		lazy.tasksWidget.AddItem(*task.TaskArn, *task.TaskArn, 0, func() {})
	}
	return nil
}

func (lazy *Lazy) handleTaskSelection(task ecsTypes.Task) {
	lazy.log("Task %s selected\n", *task.TaskArn)
	lazy.containersWidget.Clear()
	for _, container := range task.Containers {
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

		lazy.app.SetInputCapture(func(event *tcell.EventKey) *tcell.EventKey {
			switch event.Rune() {
			case '1':
				app.SetFocus(clustersWidget)
				return nil
			case '2':
				app.SetFocus(servicesWidget)
				return nil
			case '3':
				app.SetFocus(tasksWidget)
				return nil
			case '4':
				app.SetFocus(containersWidget)
				return nil
			case '5':
				app.SetFocus(main)
				return nil
			case '6':
				app.SetFocus(logs)
				return nil
			case 'f':
				//TODO list all available tabs inside the main widget
				return nil
			case 't':
				if lazy.expandMainWidget {
					left.Clear()
					left.AddItem(main, 0, 1, false)

					root.Clear()
					root.AddItem(left, 0, 1, false)
					lazy.expandMainWidget = false
				} else {
					left.Clear()
					left.AddItem(main, 0, 4, false)
					left.AddItem(logs, 0, 1, false)

					root.Clear()
					root.AddItem(right, 0, 1, false)
					root.AddItem(left, 0, 2, false)
					lazy.expandMainWidget = true
				}
				return nil
			}
			return event
		})

		lazy.clustersWidget.SetChangedFunc(
			func(index int, mainText, secondaryText string, shortcut rune) {
				lazy.cluster = &lazy.clusters[index]
				err := lazy.handleClusterSelection(context.TODO(), *lazy.cluster)
				if err != nil {
					lazy.log("TODO %s: %v", *lazy.cluster.ClusterArn, err)
				}
			},
		)

		lazy.servicesWidget.SetChangedFunc(
			func(index int, mainText, secondaryText string, shortcut rune) {
				lazy.service = &lazy.servicesByTask[*lazy.cluster.ClusterArn][index]
				err := lazy.handleServiceSelection(context.TODO(), *lazy.service)
				if err != nil {
					lazy.log("TODO %s: %v", *lazy.service.ServiceArn, err)
				}
			},
		)
		lazy.servicesWidget.SetInputCapture(func(event *tcell.EventKey) *tcell.EventKey {
			selectedService := *lazy.service
			switch event.Rune() {
			case 'd':
				lazy.log("Describing service %s\n", *selectedService.ServiceArn)
				definitionWidget := tview.NewTextView()
				lazy.main.AddAndSwitchToPage(
					fmt.Sprintf("def:%s", *selectedService.ServiceArn),
					definitionWidget,
					true,
				)
				content, err := json.MarshalIndent(selectedService, "", "  ")
				if err != nil {
					panic(err)
				}
				definitionWidget.SetText(string(content))
			case 'l':
				lazy.log("Tailing logs for service %s\n", *selectedService.ServiceArn)
				go func() {
					err := lazy.tailServiceLogs(context.TODO(), selectedService)
					if err != nil {
						lazy.log(
							"Error tailing logs for service %s: %v\n",
							*selectedService.ServiceArn,
							err,
						)
					}
				}()
				return nil
			}
			return event
		})

		lazy.tasksWidget.SetChangedFunc(
			func(index int, mainText, secondaryText string, shortcut rune) {
				lazy.task = &lazy.tasksByService[*lazy.service.ServiceArn][index]
				lazy.handleTaskSelection(*lazy.task)
			},
		)
		lazy.tasksWidget.SetInputCapture(func(event *tcell.EventKey) *tcell.EventKey {
			if event.Rune() == 'l' {
				lazy.log("Tailing logs for task %s\n", *lazy.task.TaskArn)
				go func() {
					currentTask := *lazy.task
					err := lazy.tailTaskLogs(context.TODO(), currentTask)
					if err != nil {
						lazy.log("Error tailing logs for task %s: %v\n", *currentTask.TaskArn, err)
					}
				}()
				return nil
			}
			return event
		})

		lazy.containersWidget.SetChangedFunc(
			func(index int, mainText, secondaryText string, shortcut rune) {
				lazy.container = &lazy.task.Containers[index]
			},
		)
		lazy.containersWidget.SetInputCapture(func(event *tcell.EventKey) *tcell.EventKey {
			if event.Rune() == 'l' {
				lazy.log("Tailing logs for container %s\n", *lazy.container.ContainerArn)
				go func() {
					err := lazy.tailContainerLogs(context.TODO(), *lazy.task, *lazy.container)
					if err != nil {
						lazy.log(
							"Error tailing logs for container %s: %v",
							*lazy.container.ContainerArn,
							err,
						)
					}
				}()
				return nil
			}
			return event
		})

		go func() {
			err := lazy.loadClusters(context.TODO())
			if err != nil {
				lazy.log("Error loading clusters: %s\n", err)
			}
		}()

		if err := app.SetRoot(root, true).Run(); err != nil {
			return err
		}

		return nil
	},
}

func (lazy *Lazy) loadClusters(ctx context.Context) error {
	lazy.log("Loading clusters\n")
	listedClusters, err := lazy.ecs.ListClusters(ctx, &ecs.ListClustersInput{})
	if err != nil {
		return err
	}
	describedClusters, err := lazy.ecs.DescribeClusters(
		ctx,
		&ecs.DescribeClustersInput{
			Clusters: listedClusters.ClusterArns,
		},
	)
	if err != nil {
		return err
	}
	lazy.clusters = describedClusters.Clusters
	lazy.app.QueueUpdateDraw(func() {
		lazy.clustersWidget.Clear()
		for _, cluster := range lazy.clusters {
			lazy.clustersWidget.AddItem(
				*cluster.ClusterName,
				*cluster.ClusterArn,
				0,
				func() {},
			)
		}
		lazy.app.SetFocus(lazy.clustersWidget)
	})
	return nil
}

func (lazy *Lazy) tailServiceLogs(ctx context.Context, service ecsTypes.Service) error {
	describedTaskDefinition, err := lazy.ecs.DescribeTaskDefinition(
		ctx,
		&ecs.DescribeTaskDefinitionInput{
			TaskDefinition: service.TaskDefinition,
		},
	)
	if err != nil {
		return err
	}
	var logGroupArns []string
	for _, containerDefinition := range describedTaskDefinition.TaskDefinition.ContainerDefinitions {
		logGroupName := containerDefinition.LogConfiguration.Options["awslogs-group"]
		describedLogGroups, err := lazy.cwl.DescribeLogGroups(
			ctx,
			&cwl.DescribeLogGroupsInput{
				LogGroupNamePrefix: &logGroupName,
			},
		)
		if err != nil {
			return err
		}
		logGroups := describedLogGroups.LogGroups
		if len(logGroups) != 1 {
			return fmt.Errorf(
				"expected exactly one log group for %s, got %d",
				logGroupName,
				len(logGroups),
			)
		}
		logGroupArns = append(logGroupArns, *logGroups[0].LogGroupArn)
	}

	logsWidget := tview.NewTextView()
	lazy.main.AddAndSwitchToPage(fmt.Sprintf("logs:%s", *service.ServiceArn), logsWidget, true)

	err = startLiveTail(
		ctx,
		*lazy.cwl,
		logGroupArns,
		nil,
		Handlers{
			start: func() {
				lazy.app.QueueUpdateDraw(func() {
					_, err = fmt.Fprintf(
						logsWidget,
						"Session started for service %s\n",
						*service.ServiceArn,
					)
					if err != nil {
						lazy.log("Error writing to logs: %v", err)
					}
				})
			},
			update: func(logEvent cwlTypes.LiveTailSessionLogEvent) {
				timestamp := time.UnixMilli(*logEvent.Timestamp)
				lazy.app.QueueUpdateDraw(func() {
					_, err = fmt.Fprintf(
						logsWidget,
						"%v %s %s\n",
						timestamp,
						*logEvent.Message,
						*logEvent.LogStreamName,
					)
					if err != nil {
						lazy.log("Error writing to logs: %v", err)
					}
					logsWidget.ScrollToEnd()
				})
			},
		},
	)
	if err != nil {
		return err
	}
	return nil
}

func (lazy *Lazy) tailTaskLogs(ctx context.Context, task ecsTypes.Task) error {
	describedTaskDefinition, err := lazy.ecs.DescribeTaskDefinition(
		ctx,
		&ecs.DescribeTaskDefinitionInput{
			TaskDefinition: task.TaskDefinitionArn,
		},
	)
	if err != nil {
		return err
	}

	// TODO: Check if a page already exists
	logsWidget := tview.NewTextView()
	lazy.main.AddAndSwitchToPage(
		fmt.Sprintf("logs:%s", *task.TaskArn),
		logsWidget,
		true,
	)

	for _, containerDefinition := range describedTaskDefinition.TaskDefinition.ContainerDefinitions {
		logOptions := containerDefinition.LogConfiguration.Options
		logGroupName := logOptions["awslogs-group"]
		describedLogGroups, err := lazy.cwl.DescribeLogGroups(
			ctx,
			&cwl.DescribeLogGroupsInput{
				LogGroupNamePrefix: &logGroupName,
			},
		)
		if err != nil {
			return err
		}
		logGroups := describedLogGroups.LogGroups
		if len(logGroups) != 1 {
			return fmt.Errorf(
				"expected exactly one log group for %s, got %d",
				logGroupName,
				len(logGroups),
			)
		}
		taskFragments := strings.Split(*task.TaskArn, "/")
		taskId := taskFragments[len(taskFragments)-1]
		logStreamName := fmt.Sprintf(
			"%s/%s/%s",
			logOptions["awslogs-stream-prefix"],
			*containerDefinition.Name,
			taskId,
		)

		go func() {
			err := startLiveTail(
				ctx,
				*lazy.cwl,
				[]string{*logGroups[0].LogGroupArn},
				[]string{logStreamName},
				Handlers{
					start: func() {
						lazy.app.QueueUpdateDraw(func() {
							_, err = fmt.Fprintf(
								logsWidget,
								"Session started for container %s in task %s\n",
								*containerDefinition.Name,
								*task.TaskArn,
							)
							if err != nil {
								lazy.log("Error writing to logs: %v", err)
							}
						})
					},
					update: func(logEvent cwlTypes.LiveTailSessionLogEvent) {
						timestamp := time.UnixMilli(*logEvent.Timestamp)
						lazy.app.QueueUpdateDraw(func() {
							_, err := fmt.Fprintf(
								logsWidget,
								"%v %s %s",
								timestamp,
								*logEvent.Message,
								*logEvent.LogStreamName,
							)
							if err != nil {
								lazy.log(
									"Error writing to logs widget for container %s: %v",
									*containerDefinition.Name,
									err,
								)
							}
							logsWidget.ScrollToEnd()
						})
					},
				},
			)
			if err != nil {
				lazy.log("Error tailing logs for container %s: %v", *containerDefinition.Name, err)
			}
		}()
	}
	return nil
}

func (lazy *Lazy) tailContainerLogs(
	ctx context.Context,
	task ecsTypes.Task,
	container ecsTypes.Container,
) error {
	describedTaskDefinition, err := lazy.ecs.DescribeTaskDefinition(
		ctx,
		&ecs.DescribeTaskDefinitionInput{
			TaskDefinition: task.TaskDefinitionArn,
		},
	)
	if err != nil {
		return err
	}

	var selectedContainerDefinition ecsTypes.ContainerDefinition
	for _, containerDefinition := range describedTaskDefinition.TaskDefinition.ContainerDefinitions {
		if *containerDefinition.Name == *container.Name {
			selectedContainerDefinition = containerDefinition
			break
		}
	}

	// TODO: check if selectedContainerDefinition is not null

	logsWidget := tview.NewTextView()
	lazy.main.AddAndSwitchToPage(
		fmt.Sprintf("logs:%s:%s", *task.TaskArn, *container.Name),
		logsWidget,
		true,
	)
	logOptions := selectedContainerDefinition.LogConfiguration.Options
	// TODO: handle case when key is not present
	logGroupName := logOptions["awslogs-group"]
	describedLogGroups, err := lazy.cwl.DescribeLogGroups(
		ctx,
		&cwl.DescribeLogGroupsInput{
			LogGroupNamePrefix: &logGroupName,
		},
	)
	if err != nil {
		return err
	}

	logGroups := describedLogGroups.LogGroups
	if len(logGroups) != 1 {
		return fmt.Errorf(
			"expected exactly one log group for %s, got %d",
			logGroupName,
			len(logGroups),
		)
	}

	taskFragments := strings.Split(*task.TaskArn, "/")
	taskId := taskFragments[len(taskFragments)-1]
	logStreamName := fmt.Sprintf(
		"%s/%s/%s",
		logOptions["awslogs-stream-prefix"],
		*container.Name,
		taskId,
	)

	err = startLiveTail(
		ctx,
		*lazy.cwl,
		[]string{*logGroups[0].LogGroupArn},
		[]string{logStreamName},
		Handlers{
			start: func() {
				lazy.app.QueueUpdateDraw(func() {
					_, err = fmt.Fprintf(
						logsWidget,
						"Session started for container %s in task %s\n",
						*container.Name,
						*task.TaskArn,
					)
					if err != nil {
						lazy.log("Error writing to logs: %v", err)
					}
				})
			},
			update: func(logEvent cwlTypes.LiveTailSessionLogEvent) {
				timestamp := time.UnixMilli(*logEvent.Timestamp)
				lazy.app.QueueUpdateDraw(func() {
					_, err := fmt.Fprintf(
						logsWidget,
						"%v %s %s\n",
						timestamp,
						*logEvent.Message,
						*logEvent.LogStreamName,
					)
					if err != nil {
						lazy.log(
							"Error writing to logs widget for container %s: %v",
							*container.Name,
							err,
						)
					}
					logsWidget.ScrollToEnd()
				})
			},
		},
	)
	if err != nil {
		return err
	}

	return nil
}

func startLiveTail(
	ctx context.Context,
	cwlClient cwl.Client,
	groupIdentifiers []string,
	streamNames []string,
	handlers Handlers,
) error {
	startedLiveTail, err := cwlClient.StartLiveTail(
		ctx,
		&cwl.StartLiveTailInput{
			LogGroupIdentifiers: groupIdentifiers,
			LogStreamNames:      streamNames,
		},
	)
	if err != nil {
		return err
	}

	stream := startedLiveTail.GetStream()
	defer func() {
		err = stream.Close()
		if err != nil {
			// TODO: improve error handling
			panic(err)
		}
	}()

	eventsChan := stream.Events()
	for {
		event := <-eventsChan
		// TODO: manage all event types
		switch e := event.(type) {
		case *cwlTypes.StartLiveTailResponseStreamMemberSessionStart:
			handlers.start()
		case *cwlTypes.StartLiveTailResponseStreamMemberSessionUpdate:
			for _, logEvent := range e.Value.SessionResults {
				handlers.update(logEvent)
			}
		default:
			if err := stream.Err(); err != nil {
				return err
			}
		}
	}
}

func init() {
	rootCmd.AddCommand(lazyCmd)
}
