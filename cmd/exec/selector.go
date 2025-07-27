package exec

import (
	"github.com/aws/aws-sdk-go-v2/service/ecs/types"
	"github.com/charmbracelet/huh"
)

type Selector struct {
	form      *huh.Form
	cluster   *types.Cluster
	service   *types.Service
	task      *types.Task
	container *types.Container
}

func NewSelector() Selector {
	var clusterArn string
	var serviceArn string
	var taskArn string
	var containerName string
	form := huh.NewForm(
		huh.NewGroup(
			huh.NewSelect[string]().Title("Cluster").Value(&clusterArn),
			huh.NewSelect[string]().Title("Service").
				Value(&serviceArn).
				OptionsFunc(func() []huh.Option[string] {
					return nil
				}, clusterArn),
		),
		huh.NewGroup(
			huh.NewSelect[string]().Title("Task").
				Value(&taskArn).
				OptionsFunc(func() []huh.Option[string] {
					return nil
				}, serviceArn),
			huh.NewSelect[string]().Title("Container").
				Value(&containerName).
				OptionsFunc(func() []huh.Option[string] {
					return nil
				}, taskArn),
		),
	)
	return Selector{form: form}
}
