package tui

import (
	tea "github.com/charmbracelet/bubbletea"
	"github.com/charmbracelet/lipgloss"
)

var headerstyle = lipgloss.NewStyle().
	Align(lipgloss.Center).
	Padding(1).
	Border(lipgloss.NormalBorder(), false, false, true, false)

type HeaderWindow struct {
	heading string
	width   int
	height  int
}

func (h HeaderWindow) View() string {
	return h.heading
}

func (h HeaderWindow) Update(parent *GraphApp, msg tea.Msg) tea.Cmd {
	return nil
}

func (h *HeaderWindow) SetHeading(heading string) {
	h.heading = heading
}

func NewHeaderWindow(width, height int) *HeaderWindow {
	return &HeaderWindow{
		width:  width,
		height: height,
	}
}
