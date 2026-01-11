package selector

import (
	"context"
	"fmt"
	"log"
	"regexp"
	"slices"
	"strconv"
	"strings"

	"github.com/aws/aws-sdk-go-v2/service/ecs/types"
	"github.com/charmbracelet/huh"
	"github.com/charmbracelet/lipgloss"
	"github.com/sestrella/iecs/client"
)

var titleStyle = lipgloss.NewStyle().Bold(true)

type Selectors struct {
	client client.Client
	theme  huh.Theme
}

func NewSelectors(client client.Client, theme huh.Theme) Selectors {
	return Selectors{client: client, theme: theme}
}

func (s Selectors) Cluster(
	ctx context.Context,
	clusterRegex *regexp.Regexp,
) (*types.Cluster, error) {
	return Selector[types.Cluster]{
		theme: s.theme,
		lister: func() ([]string, error) {
			return s.client.ListClusters(ctx)
		},
		describer: func(arn string) ([]types.Cluster, error) {
			return s.client.DescribeClusters(ctx, []string{arn})
		},
		pickers: func(arns []string, selectedArn *string) []huh.Field {
			return []huh.Field{huh.NewSelect[string]().
				Title("Select a cluster").
				Options(
					huh.NewOptions(arns...)...,
				).
				Value(selectedArn).
				WithHeight(5),
			}
		},
		formatter: func(selectedRes *types.Cluster) Selection {
			return Selection{
				title: "Cluster:",
				value: *selectedRes.ClusterArn,
			}
		},
	}.Run(clusterRegex)
}

func (s Selectors) Service(
	ctx context.Context,
	cluster *types.Cluster,
	serviceRegex *regexp.Regexp,
) (*types.Service, error) {
	return Selector[types.Service]{
		theme: s.theme,
		lister: func() ([]string, error) {
			return s.client.ListServices(ctx, *cluster.ClusterArn)
		},
		describer: func(arn string) ([]types.Service, error) {
			return s.client.DescribeServices(
				ctx,
				*cluster.ClusterArn,
				[]string{arn},
			)
		},
		pickers: func(arns []string, selectedArn *string) []huh.Field {
			return []huh.Field{huh.NewSelect[string]().
				Title("Select a service").
				Options(huh.NewOptions(arns...)...).
				Value(selectedArn).
				WithHeight(5),
			}
		},
		formatter: func(selectedRes *types.Service) Selection {
			return Selection{
				title: "Service:",
				value: *selectedRes.ServiceArn,
			}
		},
	}.Run(serviceRegex)
}

func (s Selectors) Task(
	ctx context.Context,
	service *types.Service,
) (*types.Task, error) {
	return Selector[types.Task]{
		theme: s.theme,
		lister: func() ([]string, error) {
			return s.client.ListTasks(ctx, *service.ClusterArn, *service.ServiceArn)
		},
		describer: func(arn string) ([]types.Task, error) {
			return s.client.DescribeTasks(ctx, *service.ClusterArn, []string{arn})
		},
		pickers: func(arns []string, selectedArn *string) []huh.Field {
			return []huh.Field{huh.NewSelect[string]().
				Title("Select a task").
				Options(huh.NewOptions(arns...)...).
				Value(selectedArn).
				WithHeight(5),
			}
		},
		formatter: func(selectedRes *types.Task) Selection {
			return Selection{
				title: "Task:",
				value: *selectedRes.TaskArn,
			}
		},
	}.Run(nil)
}

func (s Selectors) Tasks(
	ctx context.Context,
	service *types.Service,
) ([]types.Task, error) {
	taskArns, err := s.client.ListTasks(ctx, *service.ClusterArn, *service.ServiceArn)
	if err != nil {
		return nil, err
	}

	var selectedTaskArns []string
	if len(taskArns) == 1 {
		log.Println("Pre-selecting the only task available")
		selectedTaskArns = append(selectedTaskArns, taskArns[0])
	} else {
		form := huh.NewForm(
			huh.NewGroup(
				huh.NewMultiSelect[string]().
					Title("Select at least one task").
					Options(huh.NewOptions(taskArns...)...).
					Value(&selectedTaskArns).
					Validate(func(s []string) error {
						if len(s) > 0 {
							return nil
						}
						return fmt.Errorf("no task selected")
					}),
			),
		).WithTheme(&s.theme)
		if err = form.Run(); err != nil {
			return nil, err
		}
	}

	fmt.Printf("%s %s\n", titleStyle.Render("Task(s):"), strings.Join(selectedTaskArns, ","))
	tasks, err := s.client.DescribeTasks(ctx, *service.ClusterArn, selectedTaskArns)
	if err != nil {
		return nil, err
	}

	if len(tasks) == 0 {
		return nil, fmt.Errorf("no tasks selected")
	}

	return tasks, nil
}

func (s Selectors) ServiceConfig(
	ctx context.Context,
	service *types.Service,
) (*client.ServiceConfig, error) {
	currentTaskDefinition, err := s.client.DescribeTaskDefinition(ctx, *service.TaskDefinition)
	if err != nil {
		return nil, err
	}

	taskDefinitionArns, err := s.client.ListTaskDefinitions(ctx, *currentTaskDefinition.Family)
	if err != nil {
		return nil, err
	}
	if len(taskDefinitionArns) == 0 {
		return nil, fmt.Errorf("no task definitions")
	}

	var taskDefinitionArn = currentTaskDefinition.TaskDefinitionArn
	var desiredCountStr = strconv.FormatInt(int64(service.DesiredCount), 10)
	form := huh.NewForm(
		huh.NewGroup(
			huh.NewSelect[string]().
				Title("Task definition").
				Options(huh.NewOptions(taskDefinitionArns...)...).
				Value(taskDefinitionArn).
				WithHeight(5),
			huh.NewInput().
				Title("Desired count").
				Value(&desiredCountStr).
				Validate(func(s string) error {
					val, err := strconv.ParseInt(s, 10, 32)
					if err != nil {
						return fmt.Errorf("invalid number")
					}
					if val < 0 {
						return fmt.Errorf("must be greater or equal to 0")
					}
					return nil
				}),
		),
	).WithTheme(&s.theme)
	if err := form.Run(); err != nil {
		return nil, err
	}

	desiredCount, err := strconv.ParseInt(desiredCountStr, 10, 32)
	if err != nil {
		return nil, err
	}

	return &client.ServiceConfig{
		TaskDefinitionArn: *taskDefinitionArn,
		DesiredCount:      int32(desiredCount),
	}, nil
}

func (s Selectors) Container(
	ctx context.Context,
	containers []types.Container,
) (*types.Container, error) {
	return Selector[types.Container]{
		theme: s.theme,
		lister: func() ([]string, error) {
			var names []string
			for _, container := range containers {
				names = append(names, *container.Name)
			}
			return names, nil
		},
		describer: func(name string) ([]types.Container, error) {
			return slices.DeleteFunc(containers, func(container types.Container) bool {
				return *container.Name != name
			}), nil
		},
		pickers: func(names []string, selectedName *string) []huh.Field {
			return []huh.Field{huh.NewSelect[string]().
				Title("Select a container").
				Options(huh.NewOptions(names...)...).
				Value(selectedName).
				WithHeight(5),
			}
		},
		formatter: func(selectedRes *types.Container) Selection {
			return Selection{
				title: "Container:",
				value: *selectedRes.Name,
			}
		},
	}.Run(nil)
}

func (s Selectors) ContainerDefinitions(
	ctx context.Context,
	taskDefinitionArn string,
) ([]types.ContainerDefinition, error) {
	taskDefinition, err := s.client.DescribeTaskDefinition(ctx, taskDefinitionArn)
	if err != nil {
		return nil, err
	}

	var containerNames []string
	for _, containerDefinition := range taskDefinition.ContainerDefinitions {
		containerNames = append(containerNames, *containerDefinition.Name)
	}

	var selectedContainerNames []string
	if len(containerNames) == 1 {
		log.Printf("Pre-selecting the only available container")
		selectedContainerNames = append(selectedContainerNames, containerNames[0])
	} else {
		form := huh.NewForm(
			huh.NewGroup(
				huh.NewMultiSelect[string]().
					Title("Select at least one container").
					Options(huh.NewOptions(containerNames...)...).
					Value(&selectedContainerNames).
					Validate(func(s []string) error {
						if len(s) > 0 {
							return nil
						}
						return fmt.Errorf("no container selected")
					}),
			),
		).WithTheme(&s.theme)
		if err = form.Run(); err != nil {
			return nil, err
		}
	}

	var selectedContainers []types.ContainerDefinition
	for _, containerDefinition := range taskDefinition.ContainerDefinitions {
		if slices.Contains(selectedContainerNames, *containerDefinition.Name) {
			selectedContainers = append(selectedContainers, containerDefinition)
		}
	}
	if len(selectedContainers) == 0 {
		return nil, fmt.Errorf("no containers selected")
	}

	fmt.Printf(
		"%s %s\n",
		titleStyle.Render("Container(s):"),
		strings.Join(selectedContainerNames, ","),
	)
	return selectedContainers, nil
}
