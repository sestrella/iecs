package cmd

import (
	_ "embed"
	"fmt"
	"sort"
	"strings"

	"github.com/charmbracelet/huh"
	"github.com/spf13/cobra"
)

var (
	themes = map[string]func() *huh.Theme{
		"base":       huh.ThemeBase,
		"base16":     huh.ThemeBase16,
		"catppuccin": huh.ThemeCatppuccin,
		"charm":      huh.ThemeCharm,
		"dracula":    huh.ThemeDracula,
	}

	themeNames []string
	themeName  string
	theme      *huh.Theme
)

var rootCmd = &cobra.Command{
	Use:   "iecs",
	Short: "An interactive CLI for ECS",
	Long:  "Performs commons tasks on ECS, such as getting remote access or viewing logs",
	PersistentPreRunE: func(cmd *cobra.Command, args []string) error {
		if themeFunc, ok := themes[themeName]; ok {
			theme = themeFunc()
			return nil
		}

		return fmt.Errorf(
			"unsupported theme \"%s\" expecting one of: %s",
			themeName,
			strings.Join(themeNames, " "),
		)
	},
	SilenceUsage: true,
}

func Execute(version string) error {
	for name := range themes {
		themeNames = append(themeNames, fmt.Sprintf("\"%s\"", name))
	}
	sort.Strings(themeNames)

	rootCmd.PersistentFlags().
		StringVarP(
			&themeName,
			"theme",
			"t",
			"charm",
			fmt.Sprintf(
				"The theme to use. Available themes are: %s",
				strings.Join(themeNames, " "),
			),
		)
	rootCmd.Version = version

	err := rootCmd.Execute()
	if err != nil {
		return err
	}

	return nil
}
