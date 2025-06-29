package ui

import (
	"github.com/aws/aws-sdk-go-v2/service/ecs/types"
	"github.com/gdamore/tcell/v2"
	"github.com/rivo/tview"
)

type ContainerWidget struct {
	*tview.List
	containers       []types.Container
	container        types.Container
	onExecuteCommand func(types.Container)
	onTailLogs       func(types.Container)
	onReload         func()
}

func NewContainerWidget(title string) *ContainerWidget {
	widget := &ContainerWidget{}

	view := tview.NewList()
	view.SetTitle(title)
	view.SetBorder(true)
	view.SetInputCapture(func(event *tcell.EventKey) *tcell.EventKey {
		switch event.Rune() {
		case 'e':
			widget.onExecuteCommand(widget.container)
			return nil
		case 'l':
			widget.onTailLogs(widget.container)
			return nil
		case 'r':
			widget.onReload()
			return nil
		}
		return event
	})

	widget.List = view
	widget.containers = make([]types.Container, 0)

	return widget
}

func (widget *ContainerWidget) GetContainer() *types.Container {
	return &widget.container
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

func (widget *ContainerWidget) SetOnExecuteCommand(
	onExecuteCommand func(container types.Container),
) {
	widget.onExecuteCommand = onExecuteCommand
}

func (widget *ContainerWidget) SetOnTailLogs(onTailLogs func(container types.Container)) {
	widget.onTailLogs = onTailLogs
}

func (widget *ContainerWidget) SetOnReload(onReload func()) {
	widget.onReload = onReload
}

func (widget *ContainerWidget) ClearContainers() {
	widget.containers = make([]types.Container, 0)
	widget.container = types.Container{}
	widget.Clear()
}
