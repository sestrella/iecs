package cmd

import (
	"context"
	"fmt"
	"log"

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
	list     list.Model
	selected item
}

func (m model) Init() tea.Cmd {
	return nil
}

func (m model) Update(msg tea.Msg) (tea.Model, tea.Cmd) {
	switch msg := msg.(type) {
	case tea.KeyMsg:
		switch msg.String() {
		case "enter":
			i, ok := m.list.SelectedItem().(item)
			if ok {
				m.selected = i
			}
			return m, tea.Quit
		case "q", "ctrl+c":
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

type item struct {
	title, desc string
}

func (m model) View() string {
	return docStyle.Render(m.list.View())
}

func (i item) Title() string       { return i.title }
func (i item) Description() string { return i.desc }
func (i item) FilterValue() string { return i.title }

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
		clusters, err := client.ListClusters(context.TODO(), &ecs.ListClustersInput{})
		if err != nil {
			log.Fatal(err)
		}

		items := []list.Item{}
		for _, arn := range clusters.ClusterArns {
			items = append(items, &item{title: arn, desc: arn})
		}

		m := model{list: list.New(items, list.NewDefaultDelegate(), 0, 0)}
		m.list.Title = "Clusters"

		p := tea.NewProgram(m, tea.WithAltScreen())
		if _, err := p.Run(); err != nil {
			log.Fatal("Error running program:", err)
		}
		fmt.Print(m.selected.title)

		// maxResults := int32(1)
		// tasks, err := client.ListTasks(context.TODO(), &ecs.ListTasksInput{
		// 	Cluster:       &args[0],
		// 	Family:        &args[1],
		// 	DesiredStatus: types.DesiredStatusRunning,
		// 	MaxResults:    &maxResults,
		// })
		// if err != nil {
		// 	log.Fatal(err)
		// }

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
