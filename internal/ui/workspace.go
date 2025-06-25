package ui

import "github.com/rivo/tview"

type WorkspaceWidget struct {
	*tview.Pages
}

func NewWorkspaceWidget() *WorkspaceWidget {
	return &WorkspaceWidget{}
}
