package ui

import (
	"github.com/aws/aws-sdk-go-v2/service/ecs/types"
	"github.com/rivo/tview"
)

type ClusterWidget struct {
	*tview.List
	clusters []types.Cluster
	cluster  *types.Cluster
}

func NewClusterWidget(title string) *ClusterWidget {
	view := tview.NewList()
	view.SetTitle(title)
	view.SetBorder(true)
	return &ClusterWidget{
		List:     view,
		clusters: make([]types.Cluster, 0),
	}
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

func (widget *ClusterWidget) GetCluster() *types.Cluster {
	return widget.cluster
}
