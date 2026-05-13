package ui

import (
	"math/rand/v2"
	"os"
	"strings"

	"github.com/charmbracelet/bubbles/v2/list"
	"github.com/charmbracelet/bubbles/v2/spinner"
	"github.com/charmbracelet/bubbles/v2/textarea"
	"github.com/charmbracelet/bubbles/v2/viewport"
	tea "github.com/charmbracelet/bubbletea/v2"
	"github.com/charmbracelet/lipgloss/v2"
	"github.com/vybraan/vyai/internal/agent"
	"github.com/vybraan/vyai/internal/providers/gemini"
)

const gap = "\n"

func NewUIModel(gs *gemini.GeminiService, workspace string, agentRunner agent.Runner) UIModel {
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

	exploreDelegate := list.NewDefaultDelegate()
	exploreDelegate.Styles.NormalTitle = lipgloss.NewStyle().Foreground(lipgloss.Color("#D9DCCF")).Padding(0, 0, 0, 2)
	exploreDelegate.Styles.NormalDesc = lipgloss.NewStyle().Foreground(lipgloss.Color("#777777")).Padding(0, 0, 0, 2)
	exploreDelegate.Styles.DimmedTitle = lipgloss.NewStyle().Foreground(lipgloss.Color("#605F6B")).Padding(0, 0, 0, 2)
	exploreDelegate.Styles.DimmedDesc = lipgloss.NewStyle().Foreground(lipgloss.Color("#4D4D4D")).Padding(0, 0, 0, 2)
	exploreDelegate.Styles.SelectedTitle = lipgloss.NewStyle().
		Border(lipgloss.NormalBorder(), false, false, false, true).
		BorderForeground(lipgloss.Color("#c3e88d")).
		Foreground(lipgloss.Color("#c3e88d")).
		Padding(0, 0, 0, 1)
	exploreDelegate.Styles.SelectedDesc = lipgloss.NewStyle().
		Border(lipgloss.NormalBorder(), false, false, false, true).
		BorderForeground(lipgloss.Color("#c3e88d")).
		Foreground(lipgloss.Color("#68FFD6")).
		Padding(0, 0, 0, 1)

	explore := list.New([]list.Item{}, exploreDelegate, 0, 0)

	tabs := []string{"Chat", "Explore", "Settings"}
	si := buildSettingsItems(gs.Config().ChatModel, gs.Config().DescriptionModel, gs.Config().ConfigFile, gs.Config().SystemPromptFile, gs.Config().DescriptionPromptFile)
	return UIModel{
		theme:         theme,
		state:         Normal,
		explore:       explore,
		settingsItems: si,
		settingsIndex: 0,
		gsService:     gs,
		textarea:      ta,
		messages:      []string{},
		err:           nil,
		spinner:       s,
		loading:       false,
		workspace:     workspace,
		agentRunner:   agentRunner,
		Tabs:          tabs,
		activeTab:     0,
	}
}

func (m UIModel) Update(msg tea.Msg) (tea.Model, tea.Cmd) {
	var (
		textareaCmd tea.Cmd
		viewportCmd tea.Cmd
		listCmd     tea.Cmd
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
		m.explore, listCmd = m.explore.Update(msg)
	case 2:
		if msg, ok := msg.(tea.KeyMsg); ok {
			switch msg.String() {
			case "up", "k":
				if m.settingsIndex > 0 {
					m.settingsIndex--
				}
			case "down", "j":
				if m.settingsIndex < len(m.settingsItems)-1 {
					m.settingsIndex++
				}
			}
		}
	}

	cmds = append(cmds, textareaCmd, viewportCmd, spinnerCmd, listCmd)

	switch msg := msg.(type) {
	case tea.WindowSizeMsg:
		m.width = msg.Width
		m.height = msg.Height

		if !m.ready {
			m.viewport = viewport.New(viewport.WithWidth(msg.Width), viewport.WithHeight(1))
			m.viewport.Style = m.theme.ViewportStyleNormal
			m.viewport.SetContent(`Welcome to vyai - cli interface for AI!`)
			m.viewport.MouseWheelEnabled = false
			m.ready = true
		} else {
			m.viewport.SetWidth(msg.Width)
		}
		m.resizeViewport()

		m.textarea.SetWidth(msg.Width)
		if len(m.messages) > 0 {
			// Wraping content before setting it.
			m.renderViewport(strings.Join(m.messages, ""))
		}
		m.viewport.GotoBottom()

		// Configs for the second Tabs

		m.refreshExploreList()

		m.explore.DisableQuitKeybindings()
		m.explore.Title = "Conversations  Enter: open  r: rename  x: delete"

		h, v := m.theme.DocStyle.GetFrameSize()
		m.explore.SetSize(msg.Width-h, msg.Height-v-lipgloss.Height(m.headerView()))
		m.refreshSettingsList()

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
			if m.activeTab == 2 {
				m.refreshSettingsList()
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
		case "r":
			if m.activeTab == 1 {
				var renameCmd tea.Cmd
				m, renameCmd = m.renameSelectedConversation()
				cmds = append(cmds, renameCmd)
				break
			}
		case "x":
			if m.activeTab == 1 {
				var deleteCmd tea.Cmd
				m, deleteCmd = m.deleteSelectedConversation()
				cmds = append(cmds, deleteCmd)
				break
			}

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
		wrapped := renderAssistantMessage(string(msg), false)
		m.messages = append(m.messages, wrapped)
		m.loading = false
		m.notice = ""
		m.resizeViewport()
		m.spinnerIndex = rand.IntN(len(spinners) - 1)
		m.resetSpinner()
		m.renderViewport(strings.Join(m.messages, "\n"))
		m.viewport.GotoBottom()
	case streamStartMsg:
		m.streaming = true
		m.partialResponse = msg.firstToken
		m.streamTokens = msg.tokens
		m.streamErr = msg.errCh
		rendered := renderMarkdown(msg.firstToken, m.width)
		if len(m.messages) > 0 {
			m.renderViewport(strings.Join(m.messages, "\n") + "\n" + rendered)
		} else {
			m.renderViewport(rendered)
		}
		m.viewport.GotoBottom()
		return m, pollStreamCmd(m)
	case streamMsg:
		m.partialResponse = string(msg)
		rendered := renderMarkdown(string(msg), m.width)
		if len(m.messages) > 0 {
			m.renderViewport(strings.Join(m.messages, "\n") + "\n" + rendered)
		} else {
			m.renderViewport(rendered)
		}
		m.viewport.GotoBottom()
		return m, pollStreamCmd(m)
	case streamEndMsg:
		if m.partialResponse != "" {
			rendered := renderMarkdown(m.partialResponse, m.width)
			wrapped := renderAssistantMessage(strings.TrimSpace(rendered), false)
			m.messages = append(m.messages, wrapped)
			m.renderViewport(strings.Join(m.messages, "\n"))
			m.viewport.GotoBottom()
		}
		m.streaming = false
		m.loading = false
		m.partialResponse = ""
		m.streamTokens = nil
		m.streamErr = nil
		m.notice = ""
		m.resizeViewport()
		m.spinnerIndex = rand.IntN(len(spinners) - 1)
		m.resetSpinner()
	case spinner.TickMsg:
		var cmd tea.Cmd
		m.spinner, cmd = m.spinner.Update(msg)

		cmds = append(cmds, cmd)

	case noticeMsg:
		m.notice = msg.text
		m.resizeViewport()
		if msg.stopLoading {
			m.loading = false
			m.streaming = false
			m.partialResponse = ""
			m.streamTokens = nil
			m.streamErr = nil
			m.spinnerIndex = rand.IntN(len(spinners) - 1)
			m.resetSpinner()
			m.renderViewport(strings.Join(m.messages, "\n"))
			m.viewport.GotoBottom()
		}
	case serviceNoticeMsg:
		m.notice = string(msg)
		cmds = append(cmds, WaitForServiceNoticeCmd(m.gsService))
	case editorMsg:
		if msg.renameConversationID != "" {
			defer os.Remove(msg.path)
			content, err := os.ReadFile(msg.path)
			if err != nil {
				return m, noticeCmd("Conversation title could not be read.", false)
			}

			title := strings.TrimSpace(string(content))
			if err := m.gsService.RenameConversation(msg.renameConversationID, title); err != nil {
				return m, noticeCmd("Conversation title could not be updated: "+summarizeUserError(err), false)
			}
			m.refreshExploreList()
			m.deleteTarget = ""
			return m, noticeCmd("Conversation renamed.", false)
		}

		if !msg.reloadConfig {
			defer os.Remove(msg.path)
			content, err := os.ReadFile(msg.path)
			if err != nil {
				return m, noticeCmd("Editor output could not be read.", false)
			}

			m.textarea.SetValue(string(content))
			break
		}

		if err := m.gsService.ReloadConfig(); err != nil {
			return m, noticeCmd("Configuration could not be reloaded: "+summarizeUserError(err), false)
		}
		m.refreshSettingsList()
		m.notice = "Settings reloaded."

	case descriptionUpdatedMsg:
		m.refreshExploreList()
		cmds = append(cmds, WaitForDescriptionUpdateCmd(m.gsService))
	case descriptionUpdatesClosedMsg:
	case serviceNoticesClosedMsg:
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

func (m *UIModel) resizeViewport() {
	if !m.ready {
		return
	}
	headerH := lipgloss.Height(m.headerView())
	footerH := lipgloss.Height(m.footerView())
	nonViewport := headerH + 3 + footerH // header + input + footer
	if m.notice != "" {
		nonViewport++ // notice text line (the "\n" prefix in noticeView is just a separator, not an extra line)
	}
	m.viewport.SetHeight(m.height - nonViewport)
}
func (m *UIModel) refreshExploreList() {
	items, err := m.gsService.GetAllConversations()
	if err == nil {
		listItems := make([]list.Item, 0, len(items))
		for _, item := range items {
			listItems = append(listItems, newConversationListItem(item.ID, item.Description, item.ChatModel, item.UpdatedAt))
		}
		m.explore.SetItems(listItems)
		m.explore.SetShowTitle(true)
		m.explore.Title = "Conversations  Enter: open  r: rename  x: delete"
	}
}

func (m *UIModel) refreshSettingsList() {
	cfg := m.gsService.Config()
	m.settingsItems = buildSettingsItems(cfg.ChatModel, cfg.DescriptionModel, cfg.ConfigFile, cfg.SystemPromptFile, cfg.DescriptionPromptFile)
	if m.settingsIndex >= len(m.settingsItems) {
		m.settingsIndex = 0
	}
}
