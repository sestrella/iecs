package cmd

import (
	"fmt"
	"sort"
	"strings"
	"testing"

	"github.com/stretchr/testify/assert"
)

func TestThemeByName(t *testing.T) {
	t.Run("Valid Themes", func(t *testing.T) {
		for name, themeFunc := range themes {
			t.Run(name, func(t *testing.T) {
				theme, err := themeByName(name)
				assert.NoError(t, err)
				assert.NotNil(t, theme)
				assert.Equal(t, themeFunc(), theme)
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
		theme, err := themeByName(invalidThemeName)

		expectedError := fmt.Sprintf(
			"unsupported theme '%s' expecting one of: %s",
			invalidThemeName,
			strings.Join(themeNames, " "),
		)

		assert.EqualError(t, err, expectedError)
		assert.Nil(t, theme)
	})
}
