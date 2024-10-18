package tui

import (
	"fmt"
	"os"
	"strconv"
	"time"

	tea "github.com/charmbracelet/bubbletea"
	"github.com/charmbracelet/lipgloss"
	zone "github.com/lrstanley/bubblezone"
	graphs "github.com/nats-io/natscli/internal/tui/graphs"
	"golang.org/x/term"
)

const (
	SELECT_SCREEN      = iota
	MULTI_GRAPH_SCREEN = iota
	FULL_SCREEN        = iota
)

var (
	itemStyle         = lipgloss.NewStyle().PaddingLeft(4)
	selectedItemStyle = lipgloss.NewStyle().PaddingLeft(2).Foreground(lipgloss.Color("170"))
)

type GraphApp struct {
	LastUpdate time.Time
	Active     string

	header      *HeaderWindow
	body        GraphAppBody
	footer      *HelpWindow
	width       int
	height      int
	zm          *zone.Manager
	updateCB    func(string)
	activeState int
}

func NewGraphApp(active string) (GraphApp, error) {
	width, height, err := term.GetSize(0)
	if err != nil {
		return GraphApp{}, fmt.Errorf("failed to get terminal dimensions: %w", err)
	}

	state := MULTI_GRAPH_SCREEN

	if active == "" {
		state = SELECT_SCREEN
	}

	return GraphApp{
		LastUpdate: time.Now(),
		Active:     "",
		header:     nil,
		body: GraphAppBody{
			focused: -1,
		},
		width:       width,
		height:      height,
		zm:          zone.New(),
		updateCB:    nil,
		activeState: state,
	}, nil
}

func (r *GraphApp) SetUpdateCB(f func(string)) {
	r.updateCB = f
}

func (r *GraphApp) SetHeader(header *HeaderWindow) {
	r.header = header

	if r.header.width == -1 {
		r.header.width = r.width
	}

	if r.header.height == -1 {
		r.header.height = r.height
	}
}

func (r *GraphApp) SetHelpWindow(helpwindow *HelpWindow) {
	r.footer = helpwindow

	if r.footer.width == -1 {
		r.footer.width = r.width
	}

	if r.footer.height == -1 {
		r.footer.height = r.height
	}
}

func (r *GraphApp) AddChart(title string, increment int) *graphs.Graph {
	// height is calcualted by subtracting static header/footer and title values
	titleWithIndex := fmt.Sprintf("%d) %s", len(r.body.graphs)+1, title)
	chart := graphs.NewGraph(titleWithIndex, r.width/2-4, (r.height-3-3-3)/3-1, increment, r.zm)
	r.body.graphs = append(r.body.graphs, chart)
	return chart
}

func (m *GraphApp) CreateMenu(items []string) {
	m.body.menu = NewMenu(items, m.width, m.height-3-3)
}

func (r GraphApp) Init() tea.Cmd {
	return tickCmd()
}

func (r GraphApp) Update(msg tea.Msg) (tea.Model, tea.Cmd) {
	switch msg := msg.(type) {
	case tea.KeyMsg:
		switch msg.String() {
		case "q", "ctrl+c":
			return r, tea.Quit
		case "s":
			r.activeState = SELECT_SCREEN
		}
		return r, r.body.Update(&r, msg)
	case tickMsg:
		if r.Active != "" {
			r.updateCB(r.Active)
		}
		return r, tickCmd()
	case tea.WindowSizeMsg:
		r.height = msg.Height
		r.width = msg.Width
		return r, r.body.Update(&r, msg)
	}

	return r, nil
}

func (b *GraphAppBody) Update(parent *GraphApp, msg tea.Msg) tea.Cmd {
	switch msg := msg.(type) {
	case tea.KeyMsg:
		switch msg.String() {
		case "down":
			b.menu.internal.CursorDown()
		case "up":
			b.menu.internal.CursorUp()
		case "enter":
			item, ok := b.menu.internal.SelectedItem().(SelectorItem)
			parent.Active = string(item)
			if ok {
				parent.activeState = MULTI_GRAPH_SCREEN
			}
			for _, graph := range b.graphs {
				graph.Clear()
			}
		case "backspace":
			b.focused = -1
			parent.activeState = MULTI_GRAPH_SCREEN
		case "1", "2", "3", "4", "5", "6":
			if parent.activeState != MULTI_GRAPH_SCREEN {
				return nil
			}
			index, err := strconv.Atoi(msg.String())
			if err != nil {
				os.Exit(1)
			}
			b.focused = index - 1
			parent.activeState = FULL_SCREEN
		}
	case tea.WindowSizeMsg:
		b.height = msg.Height
		b.width = msg.Width
	}
	return nil
}

func (r *GraphApp) SetTickerCallback(fn func(string)) {
	r.updateCB = fn
}

func (r GraphApp) View() string {
	switch r.activeState {
	case MULTI_GRAPH_SCREEN:
		kb := []string{}
		kb = append(kb, "(1-6) Select Chart")
		kb = append(kb, "(s) Select new Nats server")
		r.footer.SetContextualHelp(kb)
		r.header.SetHeading(r.body.graphs.title(r.Active))
	case FULL_SCREEN:
		kb := []string{}
		kb = append(kb, "(←) Return to previous screen")
		r.footer.SetContextualHelp(kb)
		r.header.SetHeading(r.body.graphs.title(r.Active))
	case SELECT_SCREEN:
		kb := []string{}
		kb = append(kb, "(↑) Move cursor up")
		kb = append(kb, "(↓) Move cursor down")
		kb = append(kb, "(Enter) Select Server")
		r.footer.SetContextualHelp(kb)
		r.header.SetHeading(r.body.menu.title())
	}

	header := headerstyle.
		Width(r.header.width).
		Height(r.header.height).
		Render(r.header.View())

	footer := lipgloss.NewStyle().
		Align(lipgloss.Center).
		Width(r.width).
		Height(3).
		Border(lipgloss.NormalBorder(), true, false, false, false).
		Render(r.footer.View())

	content := lipgloss.NewStyle().
		Width(r.width).
		Height(r.height - lipgloss.Height(header) - lipgloss.Height(footer)).
		Render(r.body.View(r.activeState))

	return r.zm.Scan(lipgloss.JoinVertical(lipgloss.Top, header, content, footer))
}

func (b GraphAppBody) Init() tea.Cmd {
	return nil
}

type GraphAppBody struct {
	graphs  GraphScreen
	focused int
	menu    Menu
	height  int
	width   int
}

func (b GraphAppBody) View(state int) string {
	if state == MULTI_GRAPH_SCREEN {
		rows := []string{}
		row := []string{}

		for i, c := range b.graphs {
			row = append(row, c.View(b.width/2-4, (b.height-3-3-3)/3-1))
			if i > 0 && i%2 == 1 {
				left := lipgloss.NewStyle().
					PaddingRight(1).
					Render(row[0])
				right := lipgloss.NewStyle().
					PaddingLeft(1).
					Render(row[1])

				rows = append(rows, lipgloss.JoinHorizontal(lipgloss.Top, left, right))
				row = []string{}
			}
		}

		return lipgloss.JoinVertical(lipgloss.Top, rows...)

	} else if state == FULL_SCREEN {
		active := b.graphs[b.focused]
		return active.View(b.width, b.height-3-3-3)

	} else if state == SELECT_SCREEN {
		return b.menu.internal.View()
	}

	return "error"
}

type GraphScreen []*graphs.Graph

func (GraphScreen) title(args ...any) string {
	return fmt.Sprintf("Diagnostics for server - %s", args...)
}

type tickMsg time.Time

func tickCmd() tea.Cmd {
	return tea.Tick(time.Second*1, func(t time.Time) tea.Msg {
		return tickMsg(t)
	})
}
