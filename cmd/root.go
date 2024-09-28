package cmd

import (
	"context"
	"errors"
	"fmt"
	"os"
	"strings"

	"github.com/aws/aws-sdk-go-v2/service/ecs"
	"github.com/charmbracelet/bubbles/list"
	tea "github.com/charmbracelet/bubbletea"
	"github.com/charmbracelet/lipgloss"
	"github.com/spf13/cobra"
)

var docStyle = lipgloss.NewStyle().Margin(1, 2)

var rootCmd = &cobra.Command{
	Use:   "lazy-ecs",
	Short: "A brief description of your application",
	Long:  "TODO",
}

type item struct {
	name      string
	arn       string
	runtimeId string
}

func (i item) Title() string       { return i.name }
func (i item) Description() string { return i.arn }
func (i item) FilterValue() string { return i.name }

type model struct {
	list     list.Model
	quitting bool
}

func (m *model) Init() tea.Cmd {
	return nil
}

func (m *model) Update(msg tea.Msg) (tea.Model, tea.Cmd) {
	switch msg := msg.(type) {
	case tea.KeyMsg:
		switch msg.String() {
		case "enter":
			return m, tea.Quit
		case "q", "ctrl+c":
			m.quitting = true
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

func (m *model) View() string {
	return docStyle.Render(m.list.View())
}

func selectCluster(ctx context.Context, client *ecs.Client, clusterId string) (*item, error) {
	if clusterId != "" {
		output, err := client.DescribeClusters(ctx, &ecs.DescribeClustersInput{
			Clusters: []string{clusterId},
		})
		if err != nil {
			return nil, err
		}

		if len(output.Clusters) == 0 {
			return nil, fmt.Errorf("Cluster '%s' not found", sshClusterId)
		}

		cluster := output.Clusters[0]
		return &item{
			name: *cluster.ClusterName,
			arn:  *cluster.ClusterArn,
		}, nil
	}

	output, err := client.ListClusters(ctx, &ecs.ListClustersInput{})
	if err != nil {
		return nil, err
	}
	if len(output.ClusterArns) == 0 {
		return nil, errors.New("No clusters found")
	}

	items := []list.Item{}
	for _, arn := range output.ClusterArns {
		index := strings.LastIndex(arn, "/")
		items = append(items, item{
			name: arn[index+1:],
			arn:  arn,
		})
	}
	return newSelector("Clusters", items)
}

func newSelector(title string, items []list.Item) (*item, error) {
	model := model{list: list.New(items, list.NewDefaultDelegate(), 0, 0), quitting: false}
	model.list.Title = title

	program := tea.NewProgram(&model, tea.WithAltScreen())

	if _, err := program.Run(); err != nil {
		return nil, err
	}
	if model.quitting {
		fmt.Println("Bye bye!")
		os.Exit(1)
	}

	selected := model.list.SelectedItem().(item)
	return &selected, nil
}

// Execute adds all child commands to the root command and sets flags appropriately.
// This is called by main.main(). It only needs to happen once to the rootCmd.
func Execute() {
	err := rootCmd.Execute()
	if err != nil {
		os.Exit(1)
	}
}
