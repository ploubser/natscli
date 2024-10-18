package tui

import "strings"

type HelpWindow struct {
	width              int
	height             int
	defaultHelpMessage []string
	contextHelpMessage []string
}

func (h *HelpWindow) View() string {
	helpmessage := append(h.defaultHelpMessage, h.contextHelpMessage...)
	return strings.Join(helpmessage[:], "   •   ")
}

func (h *HelpWindow) SetContextualHelp(msg []string) {
	h.contextHelpMessage = msg
}

func NewHelpWindow(width, height int, defaultHelpMessage string) *HelpWindow {
	return &HelpWindow{
		width:              width,
		height:             height,
		defaultHelpMessage: []string{defaultHelpMessage},
	}
}
