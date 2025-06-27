package cmd

import (
	"context"
	"encoding/json"
	"fmt"
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

	"github.com/sestrella/iecs/client"
	"github.com/sestrella/iecs/internal/ui"
)

type Lazy struct {
	ecs    *ecs.Client
	cwl    *cwl.Client
	client client.Client

	app              *tview.Application
	clustersWidget   *ui.ClusterWidget
	servicesWidget   *ui.ServiceWidget
	tasksWidget      *ui.TaskWidget
	containersWidget *ui.ContainerWidget
	logsWidget       *ui.LogsWidget
	main             *ui.PagesWidget

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
	lazy.logsWidget.Log("Cluster %s selected", *cluster.ClusterName)
	services, ok := lazy.servicesByTask[*cluster.ClusterArn]
	if !ok {
		lazy.logsWidget.Log("Fetching services for cluster %s", *cluster.ClusterName)
		services, err := lazy.client.DescribeServices(ctx, *cluster.ClusterArn)
		if err != nil {
			return err
		}
		lazy.servicesByTask[*lazy.cluster.ClusterArn] = services
	}
	lazy.servicesWidget.SetServices(services)
	return nil
}

func (lazy *Lazy) handleServiceSelection(ctx context.Context, service ecsTypes.Service) error {
	lazy.logsWidget.Log("Service %s selected", *service.ServiceName)
	tasks, ok := lazy.tasksByService[*service.ServiceArn]
	if !ok {
		lazy.logsWidget.Log("Fetching tasks for service %s", *service.ServiceArn)
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
	lazy.tasksWidget.SetTasks(tasks)
	return nil
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

		clustersWidget := ui.NewClusterWidget("Clusters (1)")
		servicesWidget := ui.NewServiceWidget("Services (2)")
		tasksWidget := ui.NewTaskWidget("Tasks (3)")
		containersWidget := ui.NewContainerWidget("Containers (4)")

		right := tview.NewFlex()
		right.SetDirection(tview.FlexRow)
		right.AddItem(clustersWidget, 0, 1, false)
		right.AddItem(servicesWidget, 0, 1, false)
		right.AddItem(tasksWidget, 0, 1, false)
		right.AddItem(containersWidget, 0, 1, false)

		main := ui.NewPagesWidget("Main (5)")

		logsWidget := ui.NewLogsView()
		logsWidget.SetTitle("Logs (6)")
		logsWidget.SetBorder(true)
		logsWidget.SetLogTimestamp(true)

		left := tview.NewFlex()
		left.SetDirection(tview.FlexRow)
		left.AddItem(main, 0, 4, false)
		left.AddItem(logsWidget, 0, 1, false)

		root := tview.NewFlex()
		root.SetTitle("iecs")
		root.SetBorder(true)
		root.AddItem(right, 0, 1, false)
		root.AddItem(left, 0, 2, false)

		app := tview.NewApplication()

		lazy := &Lazy{
			ecs:              ecs.NewFromConfig(cfg),
			cwl:              cwl.NewFromConfig(cfg),
			client:           client.NewClient(cfg),
			app:              app,
			clustersWidget:   clustersWidget,
			servicesWidget:   servicesWidget,
			tasksWidget:      tasksWidget,
			containersWidget: containersWidget,
			logsWidget:       logsWidget,
			main:             main,
			servicesByTask:   make(map[string][]ecsTypes.Service),
			tasksByService:   make(map[string][]ecsTypes.Task),
		}

		containersWidget.SetExecuteCommandFunc(func(container ecsTypes.Container) {
			logsWidget.Log("Executing command on container: %s", *container.ContainerArn)
		})

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
				app.SetFocus(logsWidget)
				return nil
			case 'f':
				app.SetFocus(main.ShowIndex())

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
					left.AddItem(logsWidget, 0, 1, false)

					root.Clear()
					root.AddItem(right, 0, 1, false)
					root.AddItem(left, 0, 2, false)
					lazy.expandMainWidget = true
				}
				return nil
			}
			return event
		})

		lazy.clustersWidget.SetClusterChanged(
			func(cluster ecsTypes.Cluster) {
				lazy.cluster = &cluster
				err := lazy.handleClusterSelection(context.TODO(), cluster)
				if err != nil {
					lazy.logsWidget.Log("TODO %s: %v", *cluster.ClusterArn, err)
				}
			},
		)

		lazy.servicesWidget.SetServiceChanged(
			func(service ecsTypes.Service) {
				lazy.service = &service
				err := lazy.handleServiceSelection(context.TODO(), service)
				if err != nil {
					lazy.logsWidget.Log("TODO %s: %v", *service.ServiceArn, err)
				}
			},
		)
		lazy.servicesWidget.SetInputCapture(func(event *tcell.EventKey) *tcell.EventKey {
			selectedService := *lazy.service
			switch event.Rune() {
			case 'd':
				lazy.logsWidget.Log("Describing service %s", *selectedService.ServiceArn)
				definitionWidget := tview.NewTextView()
				serviceArnPieces := strings.Split(*selectedService.ServiceArn, ":")
				title := fmt.Sprintf(
					"Describe service: %s",
					serviceArnPieces[len(serviceArnPieces)-1],
				)
				lazy.main.AddPage(
					fmt.Sprintf("describe@%s", *selectedService.ServiceArn),
					title,
					definitionWidget,
				)
				content, err := json.MarshalIndent(selectedService, "", "  ")
				if err != nil {
					panic(err)
				}
				definitionWidget.SetText(string(content))
			case 'l':
				lazy.logsWidget.Log("Tailing logs for service %s", *selectedService.ServiceArn)
				go func() {
					err := lazy.tailServiceLogs(context.TODO(), selectedService)
					if err != nil {
						lazy.logsWidget.Log(
							"Error tailing logs for service %s: %v",
							*selectedService.ServiceArn,
							err,
						)
					}
				}()
				return nil
			}
			return event
		})

		lazy.tasksWidget.SetTaskChanged(
			func(task ecsTypes.Task) {
				lazy.task = &task
				lazy.logsWidget.Log("Task %s selected", *task.TaskArn)
				lazy.containersWidget.SetContainers(task.Containers)
			},
		)
		lazy.tasksWidget.SetInputCapture(func(event *tcell.EventKey) *tcell.EventKey {
			if event.Rune() == 'l' {
				lazy.logsWidget.Log("Tailing logs for task %s", *lazy.task.TaskArn)
				go func() {
					currentTask := *lazy.task
					err := lazy.tailTaskLogs(context.TODO(), currentTask)
					if err != nil {
						lazy.logsWidget.Log(
							"Error tailing logs for task %s: %v",
							*currentTask.TaskArn,
							err,
						)
					}
				}()
				return nil
			}
			return event
		})

		lazy.containersWidget.SetContainerChanged(
			func(container ecsTypes.Container) {
				lazy.container = &container
			},
		)
		lazy.containersWidget.SetTailLogsFunc(func(container ecsTypes.Container) {
			lazy.logsWidget.Log("Tailing logs for container %s", *container.ContainerArn)
			go func() {
				err := lazy.tailContainerLogs(context.TODO(), *lazy.task, container)
				if err != nil {
					lazy.logsWidget.Log(
						"Error tailing logs for container %s: %v",
						*container.ContainerArn,
						err,
					)
				}
			}()
		})

		go func() {
			err := lazy.loadClusters(context.TODO())
			if err != nil {
				lazy.logsWidget.Log("Error loading clusters: %s", err)
			}
		}()

		if err := app.SetRoot(root, true).Run(); err != nil {
			return err
		}

		return nil
	},
}

func (lazy *Lazy) loadClusters(ctx context.Context) error {
	lazy.logsWidget.Log("Loading clusters")
	clusters, err := lazy.client.DescribeClusters(ctx)
	if err != nil {
		return err
	}
	lazy.clusters = clusters
	lazy.app.QueueUpdateDraw(func() {
		lazy.clustersWidget.SetClusters(clusters)
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

	serviceLogsWidget := ui.NewLogsView()
	pageName := fmt.Sprintf("logs@%s", *service.ServiceArn)
	serviceArnPieces := strings.Split(*service.ServiceArn, ":")
	title := fmt.Sprintf(
		"Logs for service: %s",
		serviceArnPieces[len(serviceArnPieces)-1],
	)
	lazy.main.AddPage(pageName, title, serviceLogsWidget)

	err = startLiveTail(
		ctx,
		*lazy.cwl,
		logGroupArns,
		nil,
		Handlers{
			start: func() {
				lazy.app.QueueUpdateDraw(func() {
					serviceLogsWidget.Log(
						"Session started for service %s",
						*service.ServiceArn,
					)
				})
			},
			update: func(logEvent cwlTypes.LiveTailSessionLogEvent) {
				timestamp := time.UnixMilli(*logEvent.Timestamp)
				lazy.app.QueueUpdateDraw(func() {
					serviceLogsWidget.Log(
						"%v %s %s",
						timestamp,
						*logEvent.Message,
						*logEvent.LogStreamName,
					)
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
	taskLogsWidget := ui.NewLogsView()
	pageName := fmt.Sprintf("logs@%s", *task.TaskArn)
	taskArnPieces := strings.Split(*task.TaskArn, ":")
	title := fmt.Sprintf(
		"Logs for task: %s",
		taskArnPieces[len(taskArnPieces)-1],
	)
	lazy.main.AddPage(pageName, title, taskLogsWidget)

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
							taskLogsWidget.Log(
								"Session started for container %s in task %s",
								*containerDefinition.Name,
								*task.TaskArn,
							)
						})
					},
					update: func(logEvent cwlTypes.LiveTailSessionLogEvent) {
						timestamp := time.UnixMilli(*logEvent.Timestamp)
						lazy.app.QueueUpdateDraw(func() {
							taskLogsWidget.Log(
								"%v %s %s",
								timestamp,
								*logEvent.Message,
								*logEvent.LogStreamName,
							)
						})
					},
				},
			)
			if err != nil {
				lazy.logsWidget.Log(
					"Error tailing logs for container %s: %v",
					*containerDefinition.Name,
					err,
				)
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

	containerLogsWidget := ui.NewLogsView()
	pageName := fmt.Sprintf("logs@%s@%s", *task.TaskArn, *container.Name)
	title := fmt.Sprintf("Logs for container: %s", *container.Name)
	lazy.main.AddPage(pageName, title, containerLogsWidget)
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
					containerLogsWidget.Log(
						"Session started for container %s in task %s",
						*container.Name,
						*task.TaskArn,
					)
				})
			},
			update: func(logEvent cwlTypes.LiveTailSessionLogEvent) {
				timestamp := time.UnixMilli(*logEvent.Timestamp)
				lazy.app.QueueUpdateDraw(func() {
					containerLogsWidget.Log(
						"%v %s %s",
						timestamp,
						*logEvent.Message,
						*logEvent.LogStreamName,
					)
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
