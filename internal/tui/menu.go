package tui

import (
	"fmt"
	"io"
	"strings"

	"github.com/charmbracelet/bubbles/list"
	tea "github.com/charmbracelet/bubbletea"
)

type Menu struct {
	internal list.Model
}

func NewMenu(items []string, width, height int) Menu {
	l := []list.Item{}
	for _, item := range items {
		l = append(l, SelectorItem(item))
	}

	menu := Menu{
		internal: list.New(l, itemDelegate{}, width, height),
	}
	menu.internal.SetShowHelp(false)
	menu.internal.SetShowTitle(false)
	return menu
}

func (Menu) title() string {
	return "Please select a server"
}

type SelectorItem string

func (i SelectorItem) FilterValue() string { return "" }

type itemDelegate struct{}

func (d itemDelegate) Height() int                             { return 1 }
func (d itemDelegate) Spacing() int                            { return 0 }
func (d itemDelegate) Update(_ tea.Msg, _ *list.Model) tea.Cmd { return nil }

func (d itemDelegate) Render(w io.Writer, m list.Model, index int, listItem list.Item) {
	i, ok := listItem.(SelectorItem)
	if !ok {
		return
	}

	str := fmt.Sprintf("%d. %s", index+1, i)

	fn := itemStyle.Render
	if index == m.Index() {
		fn = func(s ...string) string {
			return selectedItemStyle.Render("> " + strings.Join(s, " "))
		}
	}

	fmt.Fprint(w, fn(str))
}
