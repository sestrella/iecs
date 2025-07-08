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
		invalidThemeName := "invalid-theme"
		theme, err := themeByName(invalidThemeName)

		var themeNames []string
		for name := range themes {
			themeNames = append(themeNames, name)
		}
		sort.Strings(themeNames)

		expectedError := fmt.Sprintf(
			"unsupported theme '%s' expecting one of: %s",
			invalidThemeName,
			strings.Join(themeNames, " "),
		)

		assert.EqualError(t, err, expectedError)
		assert.Nil(t, theme)
	})
}
