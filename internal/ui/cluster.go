package ui

import (
	"github.com/aws/aws-sdk-go-v2/service/ecs/types"
	"github.com/gdamore/tcell/v2"
	"github.com/rivo/tview"
)

type ClusterWidget struct {
	*tview.List
	clusters []types.Cluster
	cluster  *types.Cluster
	onReload func()
}

func NewClusterWidget(title string) *ClusterWidget {
	widget := &ClusterWidget{}

	view := tview.NewList()
	view.SetTitle(title)
	view.SetBorder(true)
	view.SetInputCapture(func(event *tcell.EventKey) *tcell.EventKey {
		if event.Rune() == 'r' {
			widget.onReload()
			return nil
		}
		return event
	})

	widget.List = view

	return widget
}

func (widget *ClusterWidget) GetCluster() *types.Cluster {
	return widget.cluster
}

func (widget *ClusterWidget) SetClusters(clusters []types.Cluster) {
	widget.clusters = clusters
	widget.Clear()
	for _, cluster := range clusters {
		widget.AddItem(
			*cluster.ClusterName,
			*cluster.ClusterArn,
			0,
			nil,
		)
	}
}

func (widget *ClusterWidget) SetClusterChanged(f func(cluster types.Cluster)) {
	widget.SetChangedFunc(func(index int, mainText, secondaryText string, shortcut rune) {
		widget.cluster = &widget.clusters[index]
		f(widget.clusters[index])
	})
}

func (widget *ClusterWidget) SetOnReload(onReload func()) {
	widget.onReload = onReload
}
