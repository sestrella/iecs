package cmd

import (
	"fmt"
	"sort"
	"strings"
	"testing"

	"github.com/stretchr/testify/assert"
)

func TestRootCmdPersistentPreRunE(t *testing.T) {
	t.Run("Valid Themes", func(t *testing.T) {
		for name := range themes {
			t.Run(name, func(t *testing.T) {
				themeName = name
				err := rootCmd.PersistentPreRunE(rootCmd, []string{})
				assert.NoError(t, err)
			})
		}
	})

	t.Run("Invalid Theme", func(t *testing.T) {
		// Backup and restore the global themeNames variable to ensure test isolation.
		originalThemeNames := themeNames
		defer func() { themeNames = originalThemeNames }()

		// To test the error message correctly, we need to populate the global
		// themeNames slice, which is normally done in the Execute function.
		var names []string
		for name := range themes {
			names = append(names, name)
		}
		sort.Strings(names)
		themeNames = names

		invalidThemeName := "invalid-theme"
		themeName = invalidThemeName
		err := rootCmd.PersistentPreRunE(rootCmd, []string{})

		expectedError := fmt.Sprintf(
			"unsupported theme \"%s\" expecting one of: %s",
			invalidThemeName,
			strings.Join(themeNames, " "),
		)

		assert.EqualError(t, err, expectedError)
	})
}
