package ui

import (
	"github.com/charmbracelet/bubbles/v2/list"
	"github.com/charmbracelet/bubbles/v2/spinner"
	"github.com/charmbracelet/bubbles/v2/textarea"
	"github.com/charmbracelet/bubbles/v2/viewport"
	"github.com/vybraan/vyai/internal/providers/gemini"
)

type (
	errMsg    error
	statusMsg string
	editorMsg string
)

type State string

const (
	Normal State = "NORMAL"
	Insert State = "INSERT"
)

var (
	spinners = []spinner.Spinner{
		spinner.Line,
		spinner.Dot,
		spinner.MiniDot,
		spinner.Jump,
		spinner.Pulse,
		spinner.Points,
		spinner.Monkey,
	}
)

type UIModel struct {
	width        int
	height       int
	theme        Theme
	explore      list.Model
	state        State
	viewport     viewport.Model
	messages     []string
	textarea     textarea.Model
	gsService    *gemini.GeminiService
	err          error
	spinner      spinner.Model
	spinnerIndex int
	loading      bool
	Tabs         []string
	activeTab    int
	ready        bool
}
