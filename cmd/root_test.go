package cmd

import (
	"testing"

	"github.com/charmbracelet/huh"
	"github.com/stretchr/testify/assert"
)

func TestThemeByName(t *testing.T) {
	tests := []struct {
		name          string
		themeName     string
		expectedTheme *huh.Theme
		expectError   bool
	}{
		{
			name:          "Base",
			themeName:     ThemeBase,
			expectedTheme: huh.ThemeBase(),
			expectError:   false,
		},
		{
			name:          "Base16",
			themeName:     ThemeBase16,
			expectedTheme: huh.ThemeBase16(),
			expectError:   false,
		},
		{
			name:          "Catppuccin",
			themeName:     ThemeCatppuccin,
			expectedTheme: huh.ThemeCatppuccin(),
			expectError:   false,
		},
		{
			name:          "Charm",
			themeName:     ThemeCharm,
			expectedTheme: huh.ThemeCharm(),
			expectError:   false,
		},
		{
			name:          "Dracula",
			themeName:     ThemeDracula,
			expectedTheme: huh.ThemeDracula(),
			expectError:   false,
		},
		{
			name:          "Invalid",
			themeName:     "invalid",
			expectedTheme: nil,
			expectError:   true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			theme, err := themeByName(tt.themeName)
			if tt.expectError {
				assert.Error(t, err)
				assert.Nil(t, theme)
			} else {
				assert.NoError(t, err)
				assert.Equal(t, tt.expectedTheme, theme)
			}
		})
	}
}
