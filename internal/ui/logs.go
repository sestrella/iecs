package ui

import (
	"fmt"
	"time"

	"github.com/rivo/tview"
)

type LogsWidget struct {
	*tview.TextView
	logTimestamp bool
}

func NewLogsView() *LogsWidget {
	view := tview.NewTextView()
	return &LogsWidget{
		TextView:     view,
		logTimestamp: false,
	}
}

func (widget *LogsWidget) SetLogTimestamp(logTimestamp bool) {
	widget.logTimestamp = logTimestamp
}

func (widget *LogsWidget) Log(format string, a ...any) {
	var err error
	if widget.logTimestamp {
		now := time.Now().Format(time.DateTime)
		_, err = fmt.Fprintf(widget, "%s "+format+"\n", append([]any{now}, a...)...)
	} else {
		_, err = fmt.Fprintf(widget, format, a...)
	}
	if err != nil {
		// TODO: Log error to a file
		panic(err)
	}
	widget.ScrollToEnd()
}
