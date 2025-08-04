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

type Styles struct {
	Base         lipgloss.Style
	SelectedBase lipgloss.Style

	Title        lipgloss.Style
	Subtitle     lipgloss.Style
	Text         lipgloss.Style
	TextSelected lipgloss.Style
	Muted        lipgloss.Style
	Subtle       lipgloss.Style

	Success lipgloss.Style
	Error   lipgloss.Style
	Warning lipgloss.Style
	Info    lipgloss.Style

	// Inputs
	TextArea textarea.Styles
}

type Theme struct {
	Name   string
	IsDark bool


	Styles *Styles

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

	base := lipgloss.NewStyle().
		Foreground(lipgloss.Color("#D9DCCF"))

	statusBar := lipgloss.NewStyle().
		Foreground(lipgloss.Color("#C1C6B27F")).
		Background(lipgloss.Color("#353533"))

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
			BorderBottom(false).
			BorderStyle(lipgloss.RoundedBorder()).
			BorderBottomForeground(lipgloss.Color("#7aa2f7")),

		ViewportStyleInsert: lipgloss.NewStyle().
			BorderBottom(false).
			BorderStyle(lipgloss.RoundedBorder()).
			BorderBottomForeground(lipgloss.Color("#c3e88d")),

		SenderStyle: lipgloss.NewStyle().Foreground(lipgloss.Color("5")),
		Styles: &Styles{
			Base:  base,
			Muted: base.Foreground(lipgloss.Color("#858392")),
			TextArea: textarea.Styles{
				Focused: textarea.StyleState{
					Base:             base,
					Text:             base,
					LineNumber:       base.Foreground(lipgloss.Color("#605F6B")),
					CursorLine:       base,
					CursorLineNumber: base.Foreground(lipgloss.Color("#605F6B")),
					Placeholder:      base.Foreground(lipgloss.Color("#605F6B")),
					Prompt:           base.Foreground(lipgloss.Color("#68FFD6")),
				},
				Blurred: textarea.StyleState{
					Base:             base,
					Text:             base.Foreground(lipgloss.Color("#858392")),
					LineNumber:       base.Foreground(lipgloss.Color("#858392")),
					CursorLine:       base,
					CursorLineNumber: base.Foreground(lipgloss.Color("#858392")),
					Placeholder:      base.Foreground(lipgloss.Color("#605F6B")),
					Prompt:           base.Foreground(lipgloss.Color("#858392")),
				},
				Cursor: textarea.CursorStyle{
					Color: lipgloss.Color("#c3e88d"),
					Shape: tea.CursorBar,
					Blink: true,
				},
			},
		},
	}
}

func GetTheme() Theme {
	once.Do(func() {
		themeInstance = NewDefaultTheme()
	})
	return themeInstance
}
