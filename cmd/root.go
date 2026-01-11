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
	availableThemes string
	themeStr        string
	theme           *huh.Theme
	clusterStr      string
	clusterRegex    *regexp.Regexp
	serviceStr      string
	serviceRegex    *regexp.Regexp
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

		if clusterStr != "" {
			clusterRegex = regexp.MustCompile(clusterStr)
		}

		if serviceStr != "" {
			serviceRegex = regexp.MustCompile(serviceStr)
		}

		return nil
	},
	SilenceUsage: true,
}

func Execute(version string) error {
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
	rootCmd.Version = version

	if err := rootCmd.Execute(); err != nil {
		return err
	}

	return nil
}
