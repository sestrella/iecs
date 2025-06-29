package ui

import (
	"github.com/aws/aws-sdk-go-v2/service/ecs/types"
	"github.com/rivo/tview"
)

type TaskWidget struct {
	*tview.List
	tasks []types.Task
	task  *types.Task
}

func NewTaskWidget(title string) *TaskWidget {
	list := tview.NewList()
	list.SetTitle(title)
	list.SetBorder(true)

	return &TaskWidget{
		List:  list,
		tasks: make([]types.Task, 0),
	}
}

func (widget *TaskWidget) SetTasks(tasks []types.Task) {
	widget.tasks = tasks
	widget.Clear()
	for _, task := range tasks {
		widget.AddItem(*task.TaskArn, *task.TaskArn, 0, nil)
	}
}

func (widget *TaskWidget) SetTaskChanged(f func(task types.Task)) {
	widget.SetChangedFunc(func(index int, mainText, secondaryText string, shortcut rune) {
		widget.task = &widget.tasks[index]
		f(widget.tasks[index])
	})
}

func (widget *TaskWidget) GetTask() *types.Task {
	return widget.task
}

func (widget *TaskWidget) ClearTasks() {
	widget.tasks = make([]types.Task, 0)
	widget.task = nil
	widget.Clear()
}
