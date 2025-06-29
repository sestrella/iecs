package ui

import (
	"github.com/aws/aws-sdk-go-v2/service/ecs/types"
	"github.com/rivo/tview"
)

type ServiceWidget struct {
	*tview.List
	services []types.Service
	service  *types.Service
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
		widget.service = &widget.services[index]
		f(widget.services[index])
	})
}

func (widget *ServiceWidget) GetService() *types.Service {
	return widget.service
}

func (widget *ServiceWidget) ClearServices() {
	widget.services = make([]types.Service, 0)
	widget.service = nil
	widget.Clear()
}
