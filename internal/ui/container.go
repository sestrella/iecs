package ui

import (
	"github.com/aws/aws-sdk-go-v2/service/ecs/types"
	"github.com/rivo/tview"
)

type ContainerWidget struct {
	*tview.List
	containers []types.Container
}

func NewContainerWidget(title string) *ContainerWidget {
	list := tview.NewList()
	list.SetTitle(title)
	list.SetBorder(true)

	return &ContainerWidget{
		List:       list,
		containers: make([]types.Container, 0),
	}
}

func (widget *ContainerWidget) SetContainers(containers []types.Container) {
	widget.containers = containers
	widget.Clear()
	for _, container := range containers {
		widget.AddItem(
			*container.Name,
			*container.ContainerArn,
			0,
			nil,
		)
	}
}

func (widget *ContainerWidget) SetContainerChanged(f func(container types.Container)) {
	widget.SetChangedFunc(func(index int, mainText, secondaryText string, shortcut rune) {
		f(widget.containers[index])
	})
}
