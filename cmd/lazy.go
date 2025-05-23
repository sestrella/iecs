package cmd

import (
	"github.com/rivo/tview"
	"github.com/spf13/cobra"
)

// lazyCmd represents the lazy command
var lazyCmd = &cobra.Command{
	Use:   "lazy",
	Short: "A brief description of your command",
	Run: func(cmd *cobra.Command, args []string) {
		cluster := tview.NewBox()
		cluster.SetTitle("Cluster")
		cluster.SetBorder(true)

		service := tview.NewBox()
		service.SetTitle("Service")
		service.SetBorder(true)

		task := tview.NewBox()
		task.SetTitle("Task")
		task.SetBorder(true)

		right := tview.NewFlex()
		right.SetDirection(tview.FlexRow)
		right.AddItem(cluster, 0, 1, false)
		right.AddItem(service, 0, 1, false)
		right.AddItem(task, 0, 1, false)

		main := tview.NewBox()
		main.SetTitle("main")
		main.SetBorder(true)

		left := tview.NewFlex()
		left.AddItem(main, 0, 1, false)

		root := tview.NewFlex()
		root.SetTitle("iecs")
		root.SetBorder(true)
		root.AddItem(right, 0, 1, false)
		root.AddItem(left, 0, 2, false)

		app := tview.NewApplication()
		if err := app.SetRoot(root, true).Run(); err != nil {
			panic(err)
		}
	},
}

func init() {
	rootCmd.AddCommand(lazyCmd)
}
