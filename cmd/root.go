package cmd

import (
	_ "embed"
	"fmt"
	"sort"
	"strings"

	"github.com/sestrella/iecs/cmd/exec"
	"github.com/sestrella/iecs/cmd/logs"
	"github.com/sestrella/iecs/cmd/update"
	"github.com/sestrella/iecs/selector"
	"github.com/spf13/cobra"
)

var (
	availableThemes string
	theme           string
)

var rootCmd = &cobra.Command{
	Use:   "iecs",
	Short: "An interactive CLI for ECS",
	Long:  "Performs commons tasks on ECS, such as getting remote access or viewing logs",
	PersistentPreRunE: func(cmd *cobra.Command, args []string) error {
		if _, ok := selector.Themes[theme]; ok {
			return nil
		}

		return fmt.Errorf(
			"unsupported theme \"%s\" expecting one of: %s",
			theme,
			availableThemes,
		)
	},
	SilenceUsage: true,
}

func Execute(version string) error {
	var themeNames []string
	for name := range selector.Themes {
		themeNames = append(themeNames, fmt.Sprintf("\"%s\"", name))
	}
	sort.Strings(themeNames)
	availableThemes = strings.Join(themeNames, ", ")

	rootCmd.PersistentFlags().
		StringVarP(
			&theme,
			"theme",
			"t",
			"charm",
			fmt.Sprintf(
				"The theme to use. Available themes are: %s",
				availableThemes,
			),
		)
	rootCmd.PersistentFlags().String("cluster", "", "TODO")
	rootCmd.PersistentFlags().String("service", "", "TODO")
	rootCmd.Version = version

	rootCmd.AddCommand(exec.Cmd)
	rootCmd.AddCommand(logs.Cmd)
	rootCmd.AddCommand(update.Cmd)

	if err := rootCmd.Execute(); err != nil {
		return err
	}

	return nil
}
