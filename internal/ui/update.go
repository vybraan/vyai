package ui

import (
	"fmt"
	"math/rand/v2"
	"os"
	"strconv"
	"strings"

	"github.com/charmbracelet/bubbles/v2/list"
	"github.com/charmbracelet/bubbles/v2/spinner"
	"github.com/charmbracelet/bubbles/v2/textarea"
	"github.com/charmbracelet/bubbles/v2/viewport"
	tea "github.com/charmbracelet/bubbletea/v2"
	"github.com/charmbracelet/lipgloss/v2"
	"github.com/vybraan/vyai/internal/providers/gemini"
	"github.com/vybraan/vyai/internal/utils"
)

const gap = "\n"

func NewUIModel(gs *gemini.GeminiService) UIModel {
	theme := NewDefaultTheme()

	ta := textarea.New()

	ta.Styles = theme.Styles.TextArea //SetStyles(theme.TextArea)
	ta.SetPromptFunc(4, func(lineindex int) string {
		if lineindex == 0 {
			return "  > "
		}
		// if info.Focused {
		return theme.Styles.Base.Foreground(lipgloss.Color("#12C78F")).Render("::: ")
		// } else {
		// 	return theme.Styles.Muted.Render("::: ")
		// }
	})

	ta.SetHeight(3)
	ta.ShowLineNumbers = false
	ta.CharLimit = -1
	ta.VirtualCursor = true
	ta.Focus()

	ta.Placeholder = "Ask Anything..."

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
		m.width = msg.Width
		m.height = msg.Height
		headerHeight := lipgloss.Height(m.headerView())
		footerHeight := lipgloss.Height(m.footerView())
		textareaHeight := lipgloss.Height(strconv.Itoa(m.textarea.Height()))

		verticalMarginHeight := headerHeight + footerHeight + textareaHeight + textareaHeight + textareaHeight

		if !m.ready {

			m.viewport = viewport.New(viewport.WithWidth(msg.Width), viewport.WithHeight(msg.Height-verticalMarginHeight))
			m.viewport.Style = m.theme.ViewportStyleNormal
			m.viewport.SetContent(`Welcome to vyai - cli interface for AI!`)
			m.viewport.MouseWheelEnabled = false
			m.ready = true
		} else {
			m.viewport.SetWidth(msg.Width)
			m.viewport.SetHeight(msg.Height - verticalMarginHeight)

		}

		m.textarea.SetWidth(msg.Width)
		if len(m.messages) > 0 {
			// Wraping content before setting it.
			m.renderViewport(strings.Join(m.messages, ""))
		}
		m.viewport.GotoBottom()

		// Configs for the second Tabs

		m.refreshExploreList()

		m.explore.DisableQuitKeybindings()
		m.explore.Title = "vyai conversation list"

		h, v := m.theme.DocStyle.GetFrameSize()
		m.explore.SetSize(msg.Width-h, msg.Height-v-headerHeight)

	case tea.KeyMsg:
		switch msg.String() {
		case "ctrl+c":
			cmds = append(cmds, tea.Quit)
		case "tab", "ctrl+right", "shift+tab", "ctrl+left":
			m.state = Normal
			m.updateViewportStyle()

			if msg.String() == "tab" || msg.String() == "ctrl+right" {
				m.activeTab = (m.activeTab + 1) % len(m.Tabs)

			} else if msg.String() == "shift+tab" || msg.String() == "ctrl+left" {
				m.activeTab = (m.activeTab - 1 + len(m.Tabs)) % len(m.Tabs)
			}

			if m.activeTab == 1 {
				m.refreshExploreList()
			}
		case "ctrl+e":
			if m.activeTab != 0 || m.state != Insert {
				return m, nil
			}
			// m, cmd := m.openEditorForTextarea()
			_, cmd := m.openEditorForTextarea()
			// return m, cmd
			cmds = append(cmds, cmd)
		case "ctrl+n":
			if m.activeTab != 0 || m.state != Normal {
			}
			// m, cmd := m.NewConversation()
			_, cmd := m.NewConversation()
			cmds = append(cmds, cmd)

		case "enter":
			if m.loading {
				return m, nil
			}
			var enterCmd tea.Cmd
			m, enterCmd = m.handleKeyEnter()
			cmds = append(cmds, enterCmd)

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
		m.renderViewport(strings.Join(m.messages, "\n"))
		m.viewport.GotoBottom()
	case spinner.TickMsg:
		var cmd tea.Cmd
		m.spinner, cmd = m.spinner.Update(msg)

		cmds = append(cmds, cmd)

	case errMsg:
		m.err = msg
		m.loading = false
		m.spinnerIndex = rand.IntN(len(spinners) - 1)
		m.resetSpinner()
		m.renderViewport(strings.Join(m.messages, ""))
		m.viewport.GotoBottom()
	case editorMsg:

		defer os.Remove(string(msg))
		content, err := os.ReadFile(string(msg))
		if err != nil {
			return m, func() tea.Msg { return errMsg(fmt.Errorf("Error reading file")) }
		}

		m.textarea.SetValue(string(content))

	case descriptionUpdatedMsg:
		m.refreshExploreList()
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
