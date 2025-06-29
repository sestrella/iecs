package ui

import (
	"github.com/aws/aws-sdk-go-v2/service/ecs/types"
	"github.com/gdamore/tcell/v2"
	"github.com/rivo/tview"
)

type ContainerWidget struct {
	*tview.List
	containers         []types.Container
	container          types.Container
	executeCommandFunc func(types.Container)
	tailLogsFunc       func(types.Container)
}

func NewContainerWidget(title string) *ContainerWidget {
	widget := &ContainerWidget{}

	list := tview.NewList()
	list.SetTitle(title)
	list.SetBorder(true)
	list.SetInputCapture(func(event *tcell.EventKey) *tcell.EventKey {
		switch event.Rune() {
		case 'e':
			widget.executeCommandFunc(widget.container)
			return nil
		case 'l':
			widget.tailLogsFunc(widget.container)
			return nil
		}
		return event
	})
	widget.List = list

	return widget
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
		widget.container = widget.containers[index]
		f(widget.container)
	})
}

func (widget *ContainerWidget) SetExecuteCommandFunc(f func(container types.Container)) {
	widget.executeCommandFunc = f
}

func (widget *ContainerWidget) SetTailLogsFunc(f func(container types.Container)) {
	widget.tailLogsFunc = f
}

func (widget *ContainerWidget) GetContainer() *types.Container {
	return &widget.container
}
