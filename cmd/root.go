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
	awsClient       client.Client
	selectors       selector.Selectors
	clusterStr      string
	cluster         *types.Cluster
	serviceStr      string
	service         *types.Service
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

		awsClient := client.NewClient(cfg)

		if selectedTheme, ok := themes[themeStr]; ok {
			theme = selectedTheme
		} else {
			return fmt.Errorf("unsupported theme \"%s\" expecting one of: %s", themeStr, availableThemes)
		}

		selectors = selector.NewSelectors(awsClient, *theme)

		if clusterStr != "" {
			clusterRegex, err := regexp.Compile(clusterStr)
			if err != nil {
				return err
			}

			cluster, err = selectors.Cluster(context.TODO(), clusterRegex)
			if err != nil {
				return err
			}
		}

		if serviceStr != "" {
			serviceRegex, err := regexp.Compile(serviceStr)
			if err != nil {
				return err
			}

			service, err = selectors.Service(context.TODO(), cluster, serviceRegex)
			if err != nil {
				return err
			}

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
		StringVar(&clusterStr, "cluster", "", "A regex pattern for filtering clusters")
	rootCmd.PersistentFlags().
		StringVar(&serviceStr, "service", "", "A regex pattern for filtering services")
}
