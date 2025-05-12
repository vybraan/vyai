package ui

import (
	"math/rand/v2"
	"strconv"
	"strings"

	"github.com/charmbracelet/bubbles/list"
	"github.com/charmbracelet/bubbles/spinner"
	"github.com/charmbracelet/bubbles/textarea"
	"github.com/charmbracelet/bubbles/viewport"
	tea "github.com/charmbracelet/bubbletea"
	"github.com/charmbracelet/lipgloss"
	"github.com/vybraan/vyai/internal/providers/gemini"
	"github.com/vybraan/vyai/internal/utils"
)

const gap = "\n"

func NewUIModel(gs *gemini.GeminiService) UIModel {
	theme := NewDefaultTheme()

	ta := textarea.New()
	ta.Placeholder = "Send a message..."
	// ta.Focus()

	// ta.Prompt = "┃ "
	ta.Prompt = ""
	ta.CharLimit = 15000
	ta.ShowLineNumbers = true

	// ta.SetWidth(30)
	ta.SetHeight(3)

	ta.FocusedStyle.CursorLine = lipgloss.NewStyle()
	ta.ShowLineNumbers = false
	ta.KeyMap.InsertNewline.SetEnabled(false)

	s := spinner.New()
	s.Spinner = spinner.Dot
	s.Style = theme.SpinnerStyle

	explore := list.New([]list.Item{}, list.NewDefaultDelegate(), 0, 0)

	tabs := []string{"Chat", "Explore", "Settings"}
	return UIModel{
		theme:     theme,
		state:     Normal,
		explore:   explore,
		gsService: gs,
		textarea:  ta,
		messages:  []string{},
		err:       nil,
		spinner:   s,
		loading:   false,
		Tabs:      tabs,
		activeTab: 0,
	}
}

func (m UIModel) Update(msg tea.Msg) (tea.Model, tea.Cmd) {
	var (
		textareaCmd tea.Cmd
		viewportCmd tea.Cmd
		exploreCmd  tea.Cmd
		spinnerCmd  tea.Cmd
		cmds        []tea.Cmd
	)
	// Take care of tabs now
	switch m.activeTab {
	case 0:

		if m.state == Insert {
			m.textarea, textareaCmd = m.textarea.Update(msg)
		} else {
			m.viewport, viewportCmd = m.viewport.Update(msg)
		}

		if m.loading {
			spinnerCmd = m.spinner.Tick
		}
	case 1:

		m.explore, exploreCmd = m.explore.Update(msg)
	}

	cmds = append(cmds, textareaCmd, viewportCmd, spinnerCmd, exploreCmd)

	switch msg := msg.(type) {
	case tea.WindowSizeMsg:
		headerHeight := lipgloss.Height(m.headerView())
		footerHeight := lipgloss.Height(m.footerView())
		textareaHeight := lipgloss.Height(strconv.Itoa(m.textarea.Height()))

		verticalMarginHeight := headerHeight + footerHeight + textareaHeight + textareaHeight + textareaHeight

		if !m.ready {

			m.viewport = viewport.New(msg.Width, msg.Height-verticalMarginHeight)

			m.viewport.Style = lipgloss.NewStyle().BorderBottom(true).BorderStyle(lipgloss.RoundedBorder()).BorderBottomForeground(lipgloss.Color("#7aa2f7"))
			m.viewport.SetContent(`Welcome to vyai - cli interface for AI!`)
			m.viewport.MouseWheelEnabled = false
			m.ready = true
		} else {
			m.viewport.Width = msg.Width
			m.viewport.Height = msg.Height - verticalMarginHeight

		}

		m.textarea.SetWidth(msg.Width)
		if len(m.messages) > 0 {
			// Wraping content before setting it.
			m.viewport.SetContent(m.theme.DocStyle.Width(m.viewport.Width).Render(strings.Join(m.messages, "")))
		}
		m.viewport.GotoBottom()

		// Configs for the second Tabs

		m.refreshExploreList()

		m.explore.DisableQuitKeybindings()
		m.explore.Title = "vyai conversation list"

		h, v := m.theme.DocStyle.GetFrameSize()
		m.explore.SetSize(msg.Width-h, msg.Height-v-headerHeight)

	case tea.KeyMsg:
		switch msg.Type {
		case tea.KeyCtrlC:
			return m, tea.Quit
		case tea.KeyTab, tea.KeyCtrlRight, tea.KeyShiftTab, tea.KeyCtrlLeft:
			m.state = Normal
			m.updateViewportStyle()

			if msg.Type == tea.KeyTab || msg.Type == tea.KeyCtrlRight {
				m.activeTab = (m.activeTab + 1) % len(m.Tabs)

			} else if msg.Type == tea.KeyShiftTab || msg.Type == tea.KeyCtrlLeft {
				m.activeTab = (m.activeTab - 1 + len(m.Tabs)) % len(m.Tabs)
			}

			if m.activeTab == 1 {
				m.refreshExploreList()
			}
			return m, nil
		case tea.KeyCtrlE:
			if m.activeTab != 0 || m.state != Insert {
				return m, nil
			}
			m, cmd := m.openEditorForTextarea()
			return m, cmd
		case tea.KeyCtrlN:
			if m.activeTab != 0 || m.state != Normal {
				return m, nil
			}
			m, cmd := m.NewConversation()
			return m, cmd
		case tea.KeyEnter:
			if m.loading {
				return m, nil
			}
			m, enterCmd := m.handleKeyEnter()
			return m, enterCmd

		default:
			switch msg.String() {
			case "i":
				switch m.activeTab {
				case 0:
					m.state = Insert
					m.textarea.Focus()

					m.viewport.MouseWheelEnabled = false

					cmds = append(cmds, textarea.Blink)
				}

			case "esc":

				switch m.activeTab {
				case 0:
					m.viewport.MouseWheelEnabled = false
					m.state = Normal
					m.textarea.Blur()
				}
			}
			m.updateViewportStyle()

		}

	case statusMsg:
		m.messages = append(m.messages, string(msg))
		m.loading = false
		m.spinnerIndex = rand.IntN(len(spinners)-1-0) + 0
		m.resetSpinner()
		m.viewport.SetContent(m.theme.DocStyle.Width(m.viewport.Width).Render(strings.Join(m.messages, "")))
		m.viewport.GotoBottom()
	case spinner.TickMsg:
		var cmd tea.Cmd
		m.spinner, cmd = m.spinner.Update(msg)
		return m, cmd
	case errMsg:
		m.err = msg
		m.loading = false
		m.spinnerIndex = rand.IntN(len(spinners) - 1)
		m.resetSpinner()
		m.viewport.SetContent(m.theme.DocStyle.Width(m.viewport.Width).Render(strings.Join(m.messages, "\n")))
		m.viewport.GotoBottom()
		return m, nil
	}
	return m, tea.Batch(cmds...)
}

func (m *UIModel) updateViewportStyle() {
	if m.state == Insert {
		m.viewport.Style = m.theme.ViewportStyleInsert
	} else {
		m.viewport.Style = m.theme.ViewportStyleNormal
	}
}
func (m *UIModel) refreshExploreList() {
	items, err := m.gsService.GetAllConversations()
	if err == nil {
		m.explore.SetItems(utils.ConvertToItemList(items))
		m.explore.SetShowTitle(true)
	}
}
