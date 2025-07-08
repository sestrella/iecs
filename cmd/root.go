package cmd

import (
	_ "embed"
	"fmt"
	"strings"

	"github.com/charmbracelet/huh"
	"github.com/spf13/cobra"
)

const (
	ThemeBase       string = "base"
	ThemeBase16     string = "base16"
	ThemeCatppuccin string = "catppuccin"
	ThemeCharm      string = "charm"
	ThemeDracula    string = "dracula"
)

var (
	themes    = []string{ThemeBase, ThemeBase16, ThemeCatppuccin, ThemeCharm, ThemeDracula}
	themeName string
)

var rootCmd = &cobra.Command{
	Use:          "iecs",
	Short:        "An interactive CLI for ECS",
	Long:         "Performs commons tasks on ECS, such as getting remote access or viewing logs",
	SilenceUsage: true,
}

func Execute(version string) error {
	rootCmd.PersistentFlags().
		StringVarP(
			&themeName,
			"theme",
			"t",
			"charm",
			fmt.Sprintf(
				"The theme to use. Available themes are: %s",
				strings.Join(themes, " "),
			),
		)
	rootCmd.Version = version

	err := rootCmd.Execute()
	if err != nil {
		return err
	}

	return nil
}

func themeByName(themeName string) (*huh.Theme, error) {
	switch themeName {
	case ThemeBase:
		return huh.ThemeBase(), nil
	case ThemeBase16:
		return huh.ThemeBase16(), nil
	case ThemeCatppuccin:
		return huh.ThemeCatppuccin(), nil
	case ThemeCharm:
		return huh.ThemeCharm(), nil
	case ThemeDracula:
		return huh.ThemeDracula(), nil
	default:
		return nil, fmt.Errorf(
			"unsupported theme '%s' expecting one of: %s",
			themeName,
			strings.Join(themes, " "),
		)
	}
}
