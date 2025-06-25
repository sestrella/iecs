package ui

import (
	"fmt"

	"github.com/rivo/tview"
)

type LogsWidget struct {
	*tview.TextView
}

func NewLogsView(title string) *LogsWidget {
	view := tview.NewTextView()
	view.SetTitle(title)
	view.SetBorder(true)
	return &LogsWidget{TextView: view}
}

func (widget *LogsWidget) Log(format string, a ...any) {
	// TODO: Add timestamp
	_, err := fmt.Fprintf(widget, format, a...)
	if err != nil {
		// TODO: Log error to a file
		panic(err)
	}
	widget.ScrollToEnd()
}
