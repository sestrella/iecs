package ui

import (
	"github.com/aws/aws-sdk-go-v2/service/ecs/types"
	"github.com/rivo/tview"
)

type ServiceWidget struct {
	*tview.List
	services []types.Service
}

func NewServiceWidget(title string) *ServiceWidget {
	list := tview.NewList()
	list.SetTitle(title)
	list.SetBorder(true)

	return &ServiceWidget{
		List:     list,
		services: make([]types.Service, 0),
	}
}

func (widget *ServiceWidget) SetServices(services []types.Service) {
	widget.services = services
	widget.Clear()
	for _, service := range services {
		widget.AddItem(
			*service.ServiceName,
			*service.ServiceArn,
			0,
			nil,
		)
	}
}

func (widget *ServiceWidget) SetServiceChanged(f func(service types.Service)) {
	widget.SetChangedFunc(func(index int, mainText, secondaryText string, shortcut rune) {
		f(widget.services[index])
	})
}
