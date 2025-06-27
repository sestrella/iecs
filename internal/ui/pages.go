package ui

import "github.com/rivo/tview"

type PagesWidget struct {
	*tview.Pages
	index         *tview.List
	titlesPerPage map[string]string
}

func NewPagesWidget(title string) *PagesWidget {
	index := tview.NewList()

	view := tview.NewPages()
	view.SetTitle(title)
	view.SetBorder(true)
	view.AddPage("index", index, true, false)

	return &PagesWidget{
		Pages:         view,
		index:         index,
		titlesPerPage: make(map[string]string),
	}
}

func (widget *PagesWidget) ShowIndex() tview.Primitive {
	widget.index.Clear()
	for _, pageName := range widget.GetPageNames(false) {
		if pageName != "index" {
			title := widget.GetPageTitle(pageName)
			widget.index.AddItem(title, pageName, 0, func() {
				widget.SwitchToPage(pageName)
			})
		}
	}
	widget.SwitchToPage("index")
	return widget.index
}

func (widget *PagesWidget) AddPage(name string, title string, page tview.Primitive) {
	widget.titlesPerPage[name] = title
	widget.AddAndSwitchToPage(name, page, true)
}

func (widget *PagesWidget) GetPageTitle(name string) string {
	return widget.titlesPerPage[name]
}
