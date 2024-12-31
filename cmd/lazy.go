/*
Copyright © 2024 NAME HERE <EMAIL ADDRESS>
*/
package cmd

import (
	"context"
	"encoding/json"

	"github.com/aws/aws-sdk-go-v2/config"
	"github.com/aws/aws-sdk-go-v2/service/ecs"
	"github.com/rivo/tview"
	"github.com/spf13/cobra"
)

// lazyCmd represents the lazy command
var lazyCmd = &cobra.Command{
	Use:   "lazy",
	Short: "A brief description of your command",
	Run: func(cmd *cobra.Command, args []string) {
		cfg, err := config.LoadDefaultConfig(context.TODO())
		if err != nil {
			panic(err)
		}
		ecsClient := ecs.NewFromConfig(cfg)

		app := tview.NewApplication()

		logs := tview.NewTextView()
		logs.SetTitle("Log").SetBorder(true)

		clusters := tview.NewList()
		clusters.SetTitle("Clusters").SetBorder(true)

		services := tview.NewList()
		services.SetTitle("Services").SetBorder(true)

		tasks := tview.NewList()
		tasks.SetTitle("Tasks").SetBorder(true)

		listClusters, err := ecsClient.ListClusters(context.TODO(), &ecs.ListClustersInput{})
		if err != nil {
			panic(err)
		}
		describeClusters, err := ecsClient.DescribeClusters(context.TODO(), &ecs.DescribeClustersInput{
			Clusters: listClusters.ClusterArns,
		})
		if err != nil {
			panic(err)
		}

		for _, cluster := range describeClusters.Clusters {
			clusters.AddItem(*cluster.ClusterName, *cluster.ClusterArn, 'f', func() {
				listServices, err := ecsClient.ListServices(context.TODO(), &ecs.ListServicesInput{
					Cluster: cluster.ClusterArn,
				})
				if err != nil {
					panic(err)
				}
				describeServices, err := ecsClient.DescribeServices(context.TODO(), &ecs.DescribeServicesInput{
					Cluster:  cluster.ClusterArn,
					Services: listServices.ServiceArns,
				})
				if err != nil {
					panic(err)
				}
				services.Clear()
				for _, service := range describeServices.Services {
					services.AddItem(*service.ServiceName, *service.ServiceArn, 'f', func() {
						describeTaskDefinition, err := ecsClient.DescribeTaskDefinition(context.TODO(), &ecs.DescribeTaskDefinitionInput{
							TaskDefinition: service.TaskDefinition,
						})
						if err != nil {
							panic(err)
						}
						jsonTaskDefinition, err := json.MarshalIndent(describeTaskDefinition.TaskDefinition, "", "  ")
						if err != nil {
							panic(err)
						}
						logs.SetText(string(jsonTaskDefinition))

						listTasks, err := ecsClient.ListTasks(context.TODO(), &ecs.ListTasksInput{
							Cluster:     cluster.ClusterArn,
							ServiceName: service.ServiceName,
						})
						if err != nil {
							panic(err)
						}
						describeTasks, err := ecsClient.DescribeTasks(context.TODO(), &ecs.DescribeTasksInput{
							Cluster: cluster.ClusterArn,
							Tasks:   listTasks.TaskArns,
						})
						if err != nil {
							panic(err)
						}
						tasks.Clear()
						for _, task := range describeTasks.Tasks {
							tasks.AddItem(*task.TaskArn, *task.TaskArn, 'f', nil)
						}
					})
				}
			})
		}

		leftPanel := tview.NewFlex().
			SetDirection(tview.FlexRow).
			AddItem(clusters, 0, 1, true).
			AddItem(services, 0, 1, false).
			AddItem(tasks, 0, 1, false)

		root := tview.NewFlex().
			AddItem(leftPanel, 0, 1, false).
			AddItem(logs, 0, 2, false)

		app.SetRoot(root, true).EnableMouse(true)

		if err := app.Run(); err != nil {
			panic(err)
		}
	},
}

func init() {
	rootCmd.AddCommand(lazyCmd)

	// Here you will define your flags and configuration settings.

	// Cobra supports Persistent Flags which will work for this command
	// and all subcommands, e.g.:
	// lazyCmd.PersistentFlags().String("foo", "", "A help for foo")

	// Cobra supports local flags which will only run when this command
	// is called directly, e.g.:
	// lazyCmd.Flags().BoolP("toggle", "t", false, "Help message for toggle")
}
