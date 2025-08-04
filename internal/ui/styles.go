package ui

import (
	"sync"

	"github.com/charmbracelet/bubbles/v2/textarea"
	tea "github.com/charmbracelet/bubbletea/v2"
	"github.com/charmbracelet/lipgloss/v2"
)

var (
	themeInstance Theme
	once          sync.Once
)

type Theme struct {
	DocStyle            lipgloss.Style
	SpinnerStyle        lipgloss.Style
	StatusBar           lipgloss.Style
	ModeInsert          lipgloss.Style
	ModeNormal          lipgloss.Style
	BaseInsert          lipgloss.Style
	BaseNormal          lipgloss.Style
	TopStatusBar        lipgloss.Style
	BottomModelTxt      lipgloss.Style
	ViewportStyleNormal lipgloss.Style
	ViewportStyleInsert lipgloss.Style
	SenderStyle         lipgloss.Style
}

func NewDefaultTheme() Theme {
	statusBar := lipgloss.NewStyle().
		Foreground(lipgloss.AdaptiveColor{Light: "#343433", Dark: "#C1C6B27F"}).
		Background(lipgloss.AdaptiveColor{Light: "#D9DCCF", Dark: "#353533"})

	return Theme{
		DocStyle:     lipgloss.NewStyle().Margin(1, 2),
		SpinnerStyle: lipgloss.NewStyle().Foreground(lipgloss.Color("69")),

		StatusBar: statusBar,

		ModeInsert: lipgloss.NewStyle().
			Inherit(statusBar).
			Foreground(lipgloss.Color("#000000")).
			Background(lipgloss.Color("#c3e88d")).
			Padding(0, 1),

		ModeNormal: lipgloss.NewStyle().
			Inherit(statusBar).
			Foreground(lipgloss.Color("#000000")).
			Background(lipgloss.Color("#7aa2f7")).
			Padding(0, 1),

		BaseInsert: lipgloss.NewStyle().
			Inherit(statusBar).
			Foreground(lipgloss.Color("#c3e88d")).
			Padding(0, 1),

		BaseNormal: lipgloss.NewStyle().
			Inherit(statusBar).
			Foreground(lipgloss.Color("#7aa2f7")).
			Padding(0, 1),

		TopStatusBar: lipgloss.NewStyle().
			Inherit(statusBar).
			Foreground(lipgloss.Color("#FFFDF5")).
			Background(lipgloss.Color("#ff757f")).
			Padding(0, 2).
			MarginRight(1),

		BottomModelTxt: lipgloss.NewStyle().Inherit(statusBar),

		ViewportStyleNormal: lipgloss.NewStyle().
			BorderBottom(true).
			BorderStyle(lipgloss.RoundedBorder()).
			BorderBottomForeground(lipgloss.Color("#7aa2f7")),

		ViewportStyleInsert: lipgloss.NewStyle().
			BorderBottom(true).
			BorderStyle(lipgloss.RoundedBorder()).
			BorderBottomForeground(lipgloss.Color("#c3e88d")),

		SenderStyle: lipgloss.NewStyle().Foreground(lipgloss.Color("5")),
	}
}

func GetTheme() Theme {
	once.Do(func() {
		themeInstance = NewDefaultTheme()
	})
	return themeInstance
}
