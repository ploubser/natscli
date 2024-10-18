package tui

import (
	"fmt"

	"github.com/NimbleMarkets/ntcharts/linechart/streamlinechart"
	"github.com/charmbracelet/lipgloss"
	zone "github.com/lrstanley/bubblezone"
)

type Graph struct {
	title         string
	LastValue     float64
	internal      streamlinechart.Model
	displayString string
	increment     int
}

func NewGraph(title string, width, height int, increment int, zm *zone.Manager) *Graph {
	i := streamlinechart.New(width, height)
	i.SetYRange(0, 100)
	i.SetViewYRange(0, 100)
	i.SetYStep(1)
	i.DrawXYAxisAndLabel()

	return &Graph{
		title:     title,
		LastValue: 0.0,
		internal:  i,
	}
}

func (g *Graph) Push(val float64) {
	g.LastValue = val
	g.internal.Push(val)
	g.internal.Draw()
}

func (g *Graph) View(width, height int) string {
	g.internal.Resize(width, height)
	g.redraw()
	return lipgloss.JoinVertical(lipgloss.Center, g.displayString, g.internal.View())
}

func (c *Graph) UpdateTitle(args ...any) {
	c.displayString = fmt.Sprintf(c.title, args...)
}

func (c *Graph) Clear() {
	c.internal.ClearAllData()
}

func (g *Graph) redraw() {
	max := g.internal.MaxY()
	if g.LastValue >= max {
		max = g.LastValue + float64(g.increment)
		g.internal.SetYRange(0, max)
		g.internal.SetViewYRange(0, max)
	}

	g.internal.DrawXYAxisAndLabel()
	g.internal.DrawAll()
}
