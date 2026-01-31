package cmd

import (
	_ "embed"
	"fmt"
	"regexp"
	"strings"

	"github.com/charmbracelet/huh"
	"github.com/spf13/cobra"
)

var (
	availableThemes  string
	themeStr         string
	theme            *huh.Theme
	rootCluster      string
	rootClusterRegex *regexp.Regexp
	rootService      string
	rootServiceRegex *regexp.Regexp
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
		if selectedTheme, ok := themes[themeStr]; ok {
			theme = selectedTheme
		} else {
			return fmt.Errorf("unsupported theme \"%s\" expecting one of: %s", themeStr, availableThemes)
		}

		var err error

		if rootCluster != "" {
			rootClusterRegex, err = regexp.Compile(rootCluster)
			if err != nil {
				return err
			}
		}

		if rootService != "" {
			rootServiceRegex, err = regexp.Compile(rootService)
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
		StringVar(&rootCluster, "cluster", "", "A regex pattern for filtering clusters")
	rootCmd.PersistentFlags().
		StringVar(&rootService, "service", "", "A regex pattern for filtering services")
}
