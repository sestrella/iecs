package cmd

import (
	"context"
	"errors"
	"fmt"
	"os"

	"github.com/aws/aws-sdk-go-v2/service/ecs"
	"github.com/aws/aws-sdk-go-v2/service/ecs/types"
	"github.com/charmbracelet/bubbles/list"
	tea "github.com/charmbracelet/bubbletea"
	"github.com/charmbracelet/lipgloss"
	"github.com/pterm/pterm"
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

func selectCluster(ctx context.Context, client *ecs.Client, clusterId string) (*types.Cluster, error) {
	if clusterId == "" {
		clusters, err := client.ListClusters(ctx, &ecs.ListClustersInput{})
		if err != nil {
			return nil, fmt.Errorf("Error listing clusters: %w", err)
		}
		if len(clusters.ClusterArns) == 0 {
			return nil, errors.New("No clusters found")
		}
		clusterArn, err := pterm.DefaultInteractiveSelect.WithOptions(clusters.ClusterArns).Show()
		return describeCluster(ctx, client, clusterArn)
	}
	return describeCluster(ctx, client, clusterId)
}

func describeCluster(ctx context.Context, client *ecs.Client, clusterId string) (*types.Cluster, error) {
	clusters, err := client.DescribeClusters(ctx, &ecs.DescribeClustersInput{
		Clusters: []string{clusterId},
	})
	if err != nil {
		return nil, fmt.Errorf("Error describing cluster '%s': %w", clusterId, err)
	}
	if len(clusters.Clusters) == 0 {
		return nil, fmt.Errorf("Cluster '%s' not found", clusterId)
	}
	cluster := clusters.Clusters[0]
	return &cluster, nil
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

func Execute() {
	err := rootCmd.Execute()
	if err != nil {
		os.Exit(1)
	}
}
