package ui

import "github.com/rivo/tview"

type PagesWidget struct {
	*tview.Pages
	titlesPerPage map[string]string
}

func NewPagesWidget(title string) *PagesWidget {
	view := tview.NewPages()
	view.SetTitle(title)
	view.SetBorder(true)
	return &PagesWidget{
		Pages:         view,
		titlesPerPage: make(map[string]string),
	}
}

func (widget *PagesWidget) AddPage(name string, title string, page tview.Primitive) {
	widget.titlesPerPage[name] = title
	widget.AddAndSwitchToPage(name, page, true)
}

func (widget *PagesWidget) GetPageTitle(name string) string {
	return widget.titlesPerPage[name]
}
