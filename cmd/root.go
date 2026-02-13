package cmd

import (
	"context"
	_ "embed"
	"fmt"
	"regexp"
	"strings"

	"github.com/aws/aws-sdk-go-v2/config"
	"github.com/aws/aws-sdk-go-v2/service/ecs/types"
	"github.com/charmbracelet/huh"
	"github.com/sestrella/iecs/client"
	"github.com/sestrella/iecs/selector"
	"github.com/spf13/cobra"
)

var (
	availableThemes string
	themeStr        string
	theme           *huh.Theme
	rootClient      client.Client
	rootSelectors   selector.Selectors
	rootClusterStr  string
	rootCluster     *types.Cluster
	rootServiceStr  string
	rootService     *types.Service
)

var themes = map[string]*huh.Theme{
	"base":       huh.ThemeBase(),
	"base16":     huh.ThemeBase16(),
	"catppuccin": huh.ThemeCatppuccin(),
	"charm":      huh.ThemeCharm(),
	"dracula":    huh.ThemeDracula(),
}

var rootCmd = &cobra.Command{
	Use:   "iecs",
	Short: "An interactive CLI for ECS",
	Long:  "Performs commons tasks on ECS, such as getting remote access or viewing logs",
	PersistentPreRunE: func(cmd *cobra.Command, args []string) error {
		cfg, err := config.LoadDefaultConfig(context.TODO())
		if err != nil {
			return err
		}

		rootClient = client.NewClient(cfg)

		if selectedTheme, ok := themes[themeStr]; ok {
			theme = selectedTheme
		} else {
			return fmt.Errorf("unsupported theme \"%s\" expecting one of: %s", themeStr, availableThemes)
		}

		rootSelectors = selector.NewSelectors(rootClient, *theme)

		var clusterRegex *regexp.Regexp
		if rootClusterStr != "" {
			clusterRegex, err = regexp.Compile(rootClusterStr)
			if err != nil {
				return err
			}
		}

		rootCluster, err = rootSelectors.Cluster(context.TODO(), clusterRegex)
		if err != nil {
			return err
		}

		var serviceRegex *regexp.Regexp
		if rootServiceStr != "" {
			serviceRegex, err = regexp.Compile(rootServiceStr)
			if err != nil {
				return err
			}

		}

		rootService, err = rootSelectors.Service(context.TODO(), rootCluster, serviceRegex)
		if err != nil {
			return err
		}

		return nil
	},
	SilenceUsage: true,
}

func Execute(version string) error {
	rootCmd.Version = version

	if err := rootCmd.Execute(); err != nil {
		return err
	}

	return nil
}

func init() {
	themeNames := make([]string, 0, len(themes))
	for name := range themes {
		themeNames = append(themeNames, name)
	}
	availableThemes = strings.Join(themeNames, ", ")

	rootCmd.PersistentFlags().
		StringVar(
			&themeStr,
			"theme",
			"charm",
			fmt.Sprintf(
				"The theme to use. Available themes are: %s",
				availableThemes,
			),
		)
	rootCmd.PersistentFlags().
		StringVar(&rootClusterStr, "cluster", "", "A regex pattern for filtering clusters")
	rootCmd.PersistentFlags().
		StringVar(&rootServiceStr, "service", "", "A regex pattern for filtering services")
}
