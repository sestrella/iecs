package cmd

import (
	_ "embed"
	"fmt"
	"regexp"
	"sort"
	"strings"

	"github.com/sestrella/iecs/selector"
	"github.com/spf13/cobra"
)

var (
	availableThemes string
	theme           string
	clusterRegex    *regexp.Regexp
	serviceRegex    *regexp.Regexp
)

var rootCmd = &cobra.Command{
	Use:   "iecs",
	Short: "An interactive CLI for ECS",
	Long:  "Performs commons tasks on ECS, such as getting remote access or viewing logs",
	PersistentPreRunE: func(cmd *cobra.Command, args []string) error {
		if _, ok := selector.Themes[theme]; !ok {
			return fmt.Errorf(
				"unsupported theme \"%s\" expecting one of: %s",
				theme,
				availableThemes,
			)
		}

		clusterPattern, err := cmd.Flags().GetString("cluster")
		if err != nil {
			return err
		}
		if clusterPattern != "" {
			clusterRegex = regexp.MustCompile(clusterPattern)
		}

		servicePattern, err := cmd.Flags().GetString("service")
		if err != nil {
			return err
		}
		if servicePattern != "" {
			serviceRegex = regexp.MustCompile(servicePattern)
		}

		return nil
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
	rootCmd.PersistentFlags().String("cluster", "", "A regex pattern for filtering clusters")
	rootCmd.PersistentFlags().String("service", "", "A regex pattern for filtering services")
	rootCmd.Version = version

	if err := rootCmd.Execute(); err != nil {
		return err
	}

	return nil
}
