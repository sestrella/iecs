package cmd

import (
	"context"
	"fmt"
	"log"
	"strings"

	"github.com/aws/aws-sdk-go-v2/config"
	"github.com/aws/aws-sdk-go-v2/service/ecs"
	"github.com/charmbracelet/bubbles/list"
	tea "github.com/charmbracelet/bubbletea"
	"github.com/charmbracelet/lipgloss"
	"github.com/spf13/cobra"
)

var docStyle = lipgloss.NewStyle().Margin(1, 2)

var interactive bool

type model struct {
	list list.Model
}

func (m model) Init() tea.Cmd {
	return nil
}

func (m model) Update(msg tea.Msg) (tea.Model, tea.Cmd) {
	switch msg := msg.(type) {
	case tea.KeyMsg:
		switch msg.String() {
		case "enter", "q", "ctrl+c":
			return m, tea.Quit
		}
	case tea.WindowSizeMsg:
		h, v := docStyle.GetFrameSize()
		m.list.SetSize(msg.Width-h, msg.Height-v)
	}

	var cmd tea.Cmd
	m.list, cmd = m.list.Update(msg)
	return m, cmd
}

func (m model) View() string {
	return docStyle.Render(m.list.View())
}

type item struct {
	name string
	arn  string
}

func (i item) Title() string       { return i.name }
func (i item) Description() string { return i.arn }
func (i item) FilterValue() string { return i.name }

// execCmd represents the exec command
var execCmd = &cobra.Command{
	Use:   "exec",
	Short: "A brief description of your command",
	Long: `A longer description that spans multiple lines and likely contains examples
and usage of using your command. For example:

Cobra is a CLI library for Go that empowers applications.
This application is a tool to generate the needed files
to quickly create a Cobra application.`,
	Run: func(cmd *cobra.Command, args []string) {
		cfg, err := config.LoadDefaultConfig(context.TODO())
		if err != nil {
			log.Fatal(err)
		}

		client := ecs.NewFromConfig(cfg)
		cluster, err := selectCluster(client)
		if err != nil {
			log.Fatal(err)
		}

		task, err := selectTask(client, *cluster)
		if err != nil {
			log.Fatal(err)
		}

		foo, err := client.DescribeTasks(context.TODO(), &ecs.DescribeTasksInput{
			Cluster: &cluster.arn,
			Tasks:   []string{task.arn},
		})
		if err != nil {
			log.Fatal(err)
		}

		for _, bar := range foo.Tasks[0].Containers {
			fmt.Println(string(*bar.Name))
		}

		// _, err = client.ExecuteCommand(context.TODO(), &ecs.ExecuteCommandInput{
		// 	Cluster:     &args[0],
		// 	Task:        &tasks.TaskArns[0],
		// 	Interactive: interactive,
		// })
		// if err != nil {
		// 	log.Fatal(err)
		// }

		// fmt.Printf("%+v", output)
	},
}

func selectCluster(client *ecs.Client) (*item, error) {
	output, err := client.ListClusters(context.TODO(), &ecs.ListClustersInput{})
	if err != nil {
		return nil, err
	}

	items := []list.Item{}
	for _, arn := range output.ClusterArns {
		index := strings.LastIndex(arn, "/")
		items = append(items, item{name: arn[index+1:], arn: arn})
	}
	return newSelector("Cluster", items)
}

func selectTask(client *ecs.Client, cluster item) (*item, error) {
	output, err := client.ListTasks(context.TODO(), &ecs.ListTasksInput{
		Cluster: &cluster.arn,
	})
	if err != nil {
		return nil, err
	}

	items := []list.Item{}
	for _, arn := range output.TaskArns {
		index := strings.LastIndex(arn, "/")
		items = append(items, item{name: arn[index+1:], arn: arn})
	}
	return newSelector("Tasks", items)
}

func newSelector(title string, items []list.Item) (*item, error) {
	list := list.New(items, list.NewDefaultDelegate(), 0, 0)
	list.Title = title

	model := model{list: list}
	if _, err := tea.NewProgram(model).Run(); err != nil {
		return nil, err
	}

	selected := model.list.SelectedItem().(item)
	return &selected, nil
}

func init() {
	rootCmd.AddCommand(execCmd)

	// Here you will define your flags and configuration settings.

	// Cobra supports Persistent Flags which will work for this command
	// and all subcommands, e.g.:
	// execCmd.PersistentFlags().String("foo", "", "A help for foo")

	// Cobra supports local flags which will only run when this command
	// is called directly, e.g.:
	// execCmd.Flags().StringVarP(&cluster, "cluster", "c", "", "TODO")
	execCmd.Flags().BoolVarP(&interactive, "interactive", "i", true, "TODO")
}
