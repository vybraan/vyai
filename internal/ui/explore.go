package ui

import (
	"fmt"
	"io"
	"strings"

	"github.com/charmbracelet/bubbles/v2/list"
	tea "github.com/charmbracelet/bubbletea/v2"
	"github.com/charmbracelet/lipgloss/v2"
)

type exploreDelegate struct{}

func newExploreDelegate() exploreDelegate {
	return exploreDelegate{}
}

func (d exploreDelegate) Height() int  { return 2 }
func (d exploreDelegate) Spacing() int { return 1 }

func (d exploreDelegate) Update(_ tea.Msg, _ *list.Model) tea.Cmd {
	return nil
}

func (d exploreDelegate) Render(w io.Writer, m list.Model, index int, item list.Item) {
	ci, ok := item.(conversationListItem)
	if !ok {
		return
	}

	width := m.Width()
	if width < 6 {
		return
	}

	isSelected := index == m.Index()

	barStyle := lipgloss.NewStyle().Foreground(lipgloss.Color("#605F6B"))
	titleStyle := lipgloss.NewStyle().Foreground(lipgloss.Color("#D9DCCF"))
	metaStyle := lipgloss.NewStyle().Foreground(lipgloss.Color("#777777"))

	if isSelected {
		barStyle = lipgloss.NewStyle().Foreground(lipgloss.Color("#c3e88d"))
		titleStyle = lipgloss.NewStyle().Foreground(lipgloss.Color("#c3e88d")).Bold(true)
		metaStyle = lipgloss.NewStyle().Foreground(lipgloss.Color("#68FFD6"))
	}

	bar := barStyle.Render("│")
	contentWidth := width - 4 // bar(1) + left space(1) + right padding(2)

	title := truncateOrPad(ci.Title(), contentWidth)
	meta := truncateOrPad(ci.Description(), contentWidth)

	titleLine := bar + " " + titleStyle.Render(title) + "  "
	metaLine := bar + " " + metaStyle.Render(meta) + "  "

	fmt.Fprint(w, titleLine+"\n"+metaLine)
}

func truncateOrPad(s string, maxWidth int) string {
	if maxWidth < 1 {
		return ""
	}
	w := lipgloss.Width(s)
	if w > maxWidth {
		runes := []rune(s)
		for lipgloss.Width(string(runes)) > maxWidth-1 {
			runes = runes[:len(runes)-1]
		}
		return string(runes) + "…"
	}
	if w < maxWidth {
		return s + strings.Repeat(" ", maxWidth-w)
	}
	return s
}
