package ui

import (
	"github.com/charmbracelet/bubbles/list"
	"github.com/charmbracelet/bubbles/spinner"
	"github.com/charmbracelet/bubbles/textarea"
	"github.com/charmbracelet/bubbles/viewport"
	"github.com/vybraan/vyai/internal/providers/gemini"
)

type (
	errMsg    error
	statusMsg string
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
