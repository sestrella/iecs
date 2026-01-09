package selector

import (
	"fmt"
	"regexp"
	"slices"

	"github.com/charmbracelet/huh"
)

type Selector[T any] struct {
	theme     huh.Theme
	lister    func() ([]string, error)
	describer func(arn string) ([]T, error)
	pickers   func(arns []string, selectedArn *string) []huh.Field
	formatter func(selectedRes *T) Selection
}

type Selection struct {
	title string
	value string
}

func (selector Selector[T]) Run(arnRegex *regexp.Regexp) (*T, error) {
	arns, err := selector.lister()
	if err != nil {
		return nil, err
	}

	if arnRegex != nil {
		arns = slices.DeleteFunc(arns, func(arn string) bool {
			return !arnRegex.MatchString(arn)
		})
	}

	if len(arns) == 0 {
		return nil, fmt.Errorf("TODO: no resources available")
	}

	var selectedArn string
	if len(arns) == 1 {
		selectedArn = arns[0]
	} else {
		form := huh.NewForm(huh.NewGroup(selector.pickers(arns, &selectedArn)...)).WithTheme(&selector.theme)
		if err = form.Run(); err != nil {
			return nil, err
		}
	}

	res, err := selector.describer(selectedArn)
	if err != nil {
		return nil, err
	}

	if len(res) == 1 {
		selectedRes := &res[0]
		selection := selector.formatter(selectedRes)
		fmt.Printf("%s %s\n", titleStyle.Render(selection.title), selection.value)
		return selectedRes, nil
	}

	return nil, nil
}
