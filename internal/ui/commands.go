package ui

import (
	"context"
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"strings"
	"time"

	"github.com/charmbracelet/bubbles/v2/list"
	"github.com/charmbracelet/bubbles/v2/spinner"
	"github.com/charmbracelet/bubbles/v2/textarea"
	tea "github.com/charmbracelet/bubbletea/v2"
	"github.com/charmbracelet/glamour"
	"github.com/charmbracelet/lipgloss/v2"
	"github.com/vybraan/vyai/internal/agent"
	"github.com/vybraan/vyai/internal/providers/gemini"
	"github.com/vybraan/vyai/internal/utils"
)

var focusedMessageBorder = lipgloss.Border{
	Left: "▌",
}

func renderUserMessage(message string) string {
	label := lipgloss.NewStyle().Foreground(lipgloss.Color("#6B50FF")).Bold(true).Render("You")
	content := label + "\n" + message + "\n"
	style := lipgloss.NewStyle().
		Foreground(lipgloss.Color("#DFDBDD")).
		PaddingLeft(1).
		BorderLeft(true).
		BorderStyle(lipgloss.NormalBorder()).
		BorderForeground(lipgloss.Color("#6B50FF"))
	return style.Render(content)
}

func renderAssistantMessage(message string, focused bool) string {
	borderStyle := lipgloss.NormalBorder()
	if focused {
		borderStyle = focusedMessageBorder
	}
	label := lipgloss.NewStyle().Foreground(lipgloss.Color("#12C78F")).Bold(true).Render("VyAI")
	content := label + "\n" + message + "\n"
	style := lipgloss.NewStyle().
		Foreground(lipgloss.Color("#DFDBDD")).
		PaddingLeft(1).
		BorderLeft(true).
		BorderStyle(borderStyle).
		BorderForeground(lipgloss.Color("#12C78F"))
	return style.Render(content)
}

func (m *UIModel) NewConversation() (*UIModel, tea.Cmd) {

	_, err := m.gsService.GetActiveConversation()
	if err != nil {
		return m, nil
	}

	err = m.gsService.ClearConversation(context.Background())
	if err != nil {
		return m, noticeCmd(err.Error(), false)
	}

	m.messages = []string{}

	//clear viewport
	m.renderViewport("New conversation started")
	m.resetState()

	items, err := m.gsService.GetAllConversations()
	if err == nil {
		listItems := make([]list.Item, 0, len(items))
		for _, item := range items {
			listItems = append(listItems, newConversationListItem(item.ID, item.Description, item.ChatModel, item.UpdatedAt))
		}
		m.explore.SetItems(listItems)
	} else {
		return m, noticeCmd("Conversation list could not be refreshed.", false)
	}

	return m, nil

}

func (m *UIModel) openEditorForTextarea() (*UIModel, tea.Cmd) {

	temp := "vyai-conversation_*.md"
	tempFile, err := os.CreateTemp("", temp)
	if err != nil {
		return m, noticeCmd("Editor could not be opened: "+summarizeUserError(err), false)
	}

	err = os.WriteFile(tempFile.Name(), []byte(m.textarea.Value()), 0644)
	if err != nil {
		return m, noticeCmd("Editor temp file could not be prepared: "+summarizeUserError(err), false)
	}

	return m.openEditorForPath(tempFile.Name(), false)
}

func (m *UIModel) openEditorForPath(path string, reloadConfig bool) (*UIModel, tea.Cmd) {
	editor, err := findEditor()
	if err != nil {
		return m, noticeCmd(err.Error(), false)
	}

	cmd := exec.Command(editor, path)
	cmd.Dir = filepath.Dir(path)

	execCmd := tea.ExecProcess(cmd, func(err error) tea.Msg {
		if err == nil {
			return nil
		}
		return noticeMsg{text: "Editor exited with an error: " + summarizeUserError(err)}
	})

	return m, tea.Sequence(
		execCmd,
		func() tea.Msg {
			return editorMsg{path: path, reloadConfig: reloadConfig}
		},
	)

}

func (m *UIModel) openEditorForConversationTitle(id string, title string) (*UIModel, tea.Cmd) {
	tempFile, err := os.CreateTemp("", "vyai-conversation-title_*.txt")
	if err != nil {
		return m, noticeCmd("Conversation title editor could not be opened: "+summarizeUserError(err), false)
	}

	if err := os.WriteFile(tempFile.Name(), []byte(title+"\n"), 0644); err != nil {
		return m, noticeCmd("Conversation title temp file could not be prepared: "+summarizeUserError(err), false)
	}

	editor, err := findEditor()
	if err != nil {
		return m, noticeCmd(err.Error(), false)
	}

	cmd := exec.Command(editor, tempFile.Name())
	cmd.Dir = filepath.Dir(tempFile.Name())

	execCmd := tea.ExecProcess(cmd, func(err error) tea.Msg {
		if err == nil {
			return nil
		}
		return noticeMsg{text: "Editor exited with an error: " + summarizeUserError(err)}
	})

	return m, tea.Sequence(
		execCmd,
		func() tea.Msg {
			return editorMsg{path: tempFile.Name(), renameConversationID: id}
		},
	)
}

func renderMessage(message string, focused bool) string {

	borderStyle := lipgloss.NormalBorder()
	if focused {
		borderStyle = focusedMessageBorder
	}

	style := lipgloss.NewStyle().Foreground(lipgloss.Color("#DFDBDD"))

	//message.Role == User { // check if it is user or model
	if true { // TODO: need to implement assist styles
		style = style.PaddingLeft(1).BorderLeft(true).BorderStyle(borderStyle).BorderForeground(lipgloss.Color("#6B50FF"))
	} else {
		if focused {
			style = style.PaddingLeft(1).BorderLeft(true).BorderStyle(borderStyle).BorderForeground(lipgloss.Color("#12C78F"))
		} else {
			style = style.PaddingLeft(2)
		}

	}
	joined := lipgloss.JoinVertical(lipgloss.Left, message)
	out := style.Render(joined)

	return out
}

func (m UIModel) handleKeyEnter() (UIModel, tea.Cmd) {

	switch m.activeTab {
	case 0:
		if m.state == Insert {
			prompt := m.textarea.Value()

			message := renderUserMessage(strings.TrimSpace(prompt))

			m.messages = append(m.messages, message)
			m.renderViewport(strings.Join(m.messages, "\n"))
			m.textarea.Reset()
			m.viewport.GotoBottom()

			m.loading = true
			if strings.HasPrefix(strings.TrimSpace(prompt), "/agent") {
				return m, sendAgentCmd(m, prompt)
			}
			return m, sendMessageCmd(m, prompt)
		}
	case 1:
		i, ok := m.explore.SelectedItem().(conversationListItem)
		if !ok {
			// return tea.Quit - in the future a notification
			return m, nil
		}

		if err := m.gsService.SwitchConversation(context.Background(), i.ID()); err != nil {
			m.resetState()
			m.activeTab = 0
			return m, noticeCmd("Conversation could not be opened: "+summarizeUserError(err), false)
		}
		m.messages = []string{}

		conversation, err := m.gsService.GetActiveConversation()
		if err != nil {
			m.resetState()
			m.activeTab = 0
			return m, noticeCmd("Active conversation could not be loaded: "+summarizeUserError(err), false)
		}

		messages, err := conversation.Repo.GetMessages()

		if err != nil {
			// No messages in the conversation
			m.renderViewport("chat is empty")
			m.resetState()
			m.activeTab = 0
			return m, nil
		}

		for _, message := range messages {
			if message.Role == "user" {
				text := renderUserMessage(strings.TrimSpace(message.Text))
				m.messages = append(m.messages, text)
			} else {
				rendered := renderMarkdown(message.Text, m.width)
				wrapped := renderAssistantMessage(strings.TrimSpace(rendered), false)
				m.messages = append(m.messages, wrapped)
			}

		}
		m.renderViewport(strings.Join(m.messages, "\n"))

		m.activeTab = 0
		m.resetState()

		return m, nil
	case 2:
		if m.settingsIndex >= len(m.settingsItems) {
			return m, nil
		}
		item := m.settingsItems[m.settingsIndex]
		switch item.itemType {
		case settingTypeChatModel:
			newModel := nextModel(m.gsService.Config().ChatModel)
			if err := m.gsService.SetChatModel(newModel); err != nil {
				return m, noticeCmd("Failed to update model: "+summarizeUserError(err), false)
			}
			m.refreshSettingsList()
			return m, noticeCmd("Chat model: "+newModel, false)
		case settingTypeDescModel:
			newModel := nextModel(m.gsService.Config().DescriptionModel)
			if err := m.gsService.SetDescriptionModel(newModel); err != nil {
				return m, noticeCmd("Failed to update model: "+summarizeUserError(err), false)
			}
			m.refreshSettingsList()
			return m, noticeCmd("Description model: "+newModel, false)
		default:
			_, cmd := m.openEditorForPath(item.Path(), true)
			return m, cmd
		}
	}
	return m, nil
}

func (m UIModel) renameSelectedConversation() (UIModel, tea.Cmd) {
	item, ok := m.explore.SelectedItem().(conversationListItem)
	if !ok {
		return m, nil
	}

	m.deleteTarget = ""
	_, cmd := m.openEditorForConversationTitle(item.ID(), item.Title())
	return m, cmd
}

func (m UIModel) deleteSelectedConversation() (UIModel, tea.Cmd) {
	item, ok := m.explore.SelectedItem().(conversationListItem)
	if !ok {
		return m, nil
	}

	if m.deleteTarget != item.ID() {
		m.deleteTarget = item.ID()
		return m, noticeCmd("Press x again to delete \""+item.Title()+"\".", false)
	}

	if err := m.gsService.DeleteConversation(item.ID()); err != nil {
		return m, noticeCmd("Conversation could not be deleted: "+summarizeUserError(err), false)
	}

	m.deleteTarget = ""
	m.refreshExploreList()
	return m, noticeCmd("Conversation deleted.", false)
}

func sendMessageCmd(m UIModel, prompt string) tea.Cmd {
	ctx, cancel := context.WithTimeout(context.Background(), 60*time.Second)

	tokens := make(chan string, 20)
	errCh := make(chan error, 1)

	go func() {
		defer cancel()
		_, err := m.gsService.SendMessageStream(ctx, prompt, func(token string) {
			tokens <- token
		})
		if err != nil {
			errCh <- err
		}
		close(tokens)
	}()

	return func() tea.Msg {
		select {
		case token, ok := <-tokens:
			if !ok {
				select {
				case err := <-errCh:
					return noticeMsg{text: "Request failed: " + summarizeUserError(err), stopLoading: true}
				default:
					return streamEndMsg{}
				}
			}
			return streamStartMsg{tokens: tokens, errCh: errCh, firstToken: token}
		case err := <-errCh:
			return noticeMsg{text: "Request failed: " + summarizeUserError(err), stopLoading: true}
		}
	}
}

func pollStreamCmd(m UIModel) tea.Cmd {
	return func() tea.Msg {
		select {
		case token, ok := <-m.streamTokens:
			if !ok {
				select {
				case err := <-m.streamErr:
					return noticeMsg{text: "Request failed: " + summarizeUserError(err), stopLoading: true}
				default:
					return streamEndMsg{}
				}
			}
			return streamMsg(token)
		case err := <-m.streamErr:
			return noticeMsg{text: "Request failed: " + summarizeUserError(err), stopLoading: true}
		}
	}
}

func sendAgentCmd(m UIModel, prompt string) tea.Cmd {
	return func() tea.Msg {
		userInput := strings.TrimSpace(strings.TrimPrefix(strings.TrimSpace(prompt), "/agent"))
		if m.agentRunner == nil {
			return noticeMsg{text: "Agent runner is not configured.", stopLoading: true}
		}

		output, err := m.agentRunner.Run(context.Background(), agent.RunRequest{
			Input: userInput,
			Model: m.gsService.Config().ChatModel,
		})
		if err != nil {
			return noticeMsg{text: "Agent request failed: " + summarizeUserError(err), stopLoading: true}
		}
		rendered := renderMarkdown(output, m.width)
		return statusMsg(strings.TrimSpace(rendered))
	}
}

func (m UIModel) Init() tea.Cmd {
	return tea.Batch(
		tea.EnterAltScreen,
		tea.EnableMouseAllMotion,
		tea.EnableMouseCellMotion,
		textarea.Blink,
		WaitForDescriptionUpdateCmd(m.gsService),
		WaitForServiceNoticeCmd(m.gsService),
	)
}

func (m *UIModel) resetSpinner() {

	m.spinner = spinner.New()
	m.spinner.Style = m.theme.SpinnerStyle
	m.spinner.Spinner = spinners[m.spinnerIndex]
}

func (m *UIModel) renderViewport(content string) {
	m.viewport.SetContent(m.theme.DocStyle.Width(m.viewport.Width()).Render(content))
}
func renderMarkdown(s string, width int) string {
	out, err := glamour.NewTermRenderer(
		glamour.WithAutoStyle(),
		glamour.WithWordWrap(width), // defaults to 80 - need to expand
	)
	if err != nil {
		return strings.TrimSpace(s)
	}

	renderedMessage, err := out.Render(strings.TrimSpace(s))
	if err != nil {
		return strings.TrimSpace(s)
	}

	return renderedMessage
}

func (m *UIModel) resetState() {
	m.state = Normal
	m.textarea.Reset()
	m.viewport.GotoBottom()
	m.viewport.MouseWheelEnabled = false
}

func WaitForDescriptionUpdateCmd(gsService *gemini.GeminiService) tea.Cmd {
	return func() tea.Msg {
		update, ok := <-gsService.DescriptionUpdates()
		if !ok {
			return descriptionUpdatesClosedMsg{}
		}

		return descriptionUpdatedMsg{ID: update.ID, Description: update.Description}
	}
}

func WaitForServiceNoticeCmd(gsService *gemini.GeminiService) tea.Cmd {
	return func() tea.Msg {
		notice, ok := <-gsService.Notices()
		if !ok {
			return serviceNoticesClosedMsg{}
		}

		return serviceNoticeMsg(notice.Message)
	}
}

func noticeCmd(text string, stopLoading bool) tea.Cmd {
	return func() tea.Msg {
		return noticeMsg{text: text, stopLoading: stopLoading}
	}
}

func summarizeUserError(err error) string {
	if err == nil {
		return "unknown error"
	}

	if summary, ok := utils.SummarizeKnownError(err); ok {
		return summary
	}

	return strings.TrimSpace(err.Error())
}

func findEditor() (string, error) {
	editor := os.Getenv("EDITOR")
	knownEditors := [...]string{editor, "vim", "vi", "nano", "ed"}

	for _, cmd := range knownEditors {
		if cmd == "" {
			continue
		}
		pathValue, err := exec.LookPath(cmd)
		if err != nil {
			continue
		}
		return pathValue, nil
	}

	return "", fmt.Errorf("EDITOR is not set and no fallback editor was found in PATH: %v", knownEditors[1:])
}
