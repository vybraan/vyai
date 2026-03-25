package ui

import (
	"context"
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"strings"

	"github.com/charmbracelet/bubbles/v2/list"
	"github.com/charmbracelet/bubbles/v2/spinner"
	"github.com/charmbracelet/bubbles/v2/textarea"
	tea "github.com/charmbracelet/bubbletea/v2"
	"github.com/charmbracelet/glamour"
	"github.com/charmbracelet/lipgloss/v2"
	"github.com/vybraan/vyai/internal/agent"
	"github.com/vybraan/vyai/internal/providers/gemini"
)

var focusedMessageBorder = lipgloss.Border{
	Left: "▌",
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
	editor := os.Getenv("EDITOR")

	knownEditors := [...]string{
		editor,
		"vim",
		"vi",
		"nano",
		"ed",
	}

	for _, cmd := range knownEditors {
		pathValue, err := exec.LookPath(cmd)
		if err != nil {
			continue
		}
		editor = pathValue
		break
	}

	if editor == "" {
		return m, noticeCmd(fmt.Sprintf("EDITOR is not set and no fallback editor was found in PATH: %v", knownEditors[1:]), false)
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

	editor := os.Getenv("EDITOR")
	knownEditors := [...]string{editor, "vim", "vi", "nano", "ed"}
	for _, cmd := range knownEditors {
		pathValue, err := exec.LookPath(cmd)
		if err != nil {
			continue
		}
		editor = pathValue
		break
	}

	if editor == "" {
		return m, noticeCmd(fmt.Sprintf("EDITOR is not set and no fallback editor was found in PATH: %v", knownEditors[1:]), false)
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

// renderMessage renders the provided message as a lipgloss-styled string, using a left border to indicate focus.
// When focused it uses focusedMessageBorder; otherwise it uses a normal border. The rendered output includes left
// padding and a left border with a fixed foreground color.
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

			message := renderMessage(strings.TrimSpace(prompt), false)

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
				message := renderMessage(strings.TrimSpace(message.Text), false)
				m.messages = append(m.messages, message)
			} else {
				renderedMessage := renderMarkdown(message.Text, m.width)
				m.messages = append(m.messages, strings.TrimSpace(renderedMessage))
			}

		}
		m.renderViewport(strings.Join(m.messages, "\n"))

		m.activeTab = 0
		m.resetState()

		return m, nil
	case 2:
		item, ok := m.settings.SelectedItem().(settingsItem)
		if !ok {
			return m, nil
		}
		_, cmd := m.openEditorForPath(item.Path(), true)
		return m, cmd
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

// sendMessageCmd creates a tea.Cmd that sends the provided prompt using the model's Gemini service
// and converts a successful response into a trimmed, rendered status message.
// On error it produces a noticeMsg containing a user-friendly summary of the failure and sets stopLoading to true.
func sendMessageCmd(m UIModel, prompt string) tea.Cmd {
	return func() tea.Msg {
		respChan := make(chan string)
		errChan := make(chan error)

		// send message
		go func() {
			message, err := m.gsService.SendMessage(context.Background(), prompt)
			if err != nil {
				errChan <- err
				return
			}
			respChan <- message
		}()

		select {
		case message := <-respChan:
			renderedMessage := renderMarkdown(message, m.width)
			return statusMsg(strings.TrimSpace(renderedMessage))
		case err := <-errChan:
			return noticeMsg{text: "Request failed: " + summarizeUserError(err), stopLoading: true}
		}
	}
}

// sendAgentCmd executes an agent run using the text from the provided prompt and produces a tea.Cmd that emits the agent's output or a user-facing error message.
// 
// The prompt may include a leading "/agent" prefix; that prefix and surrounding whitespace are removed before running the agent.
// If the model's agentRunner is nil, the command emits a noticeMsg with text "Agent runner is not configured." and stops loading.
// On agent execution failure the command emits a noticeMsg with text prefixed by "Agent request failed: " and a summarized error.
// On success the command emits a statusMsg containing the agent output rendered as markdown and trimmed of surrounding whitespace.
// The agent is invoked using the configured chat model from m.gsService.Config().ChatModel.
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
// renderMarkdown renders the Markdown string s into terminal-friendly ANSI-styled output using a Glamour renderer configured with automatic styling and word-wrap at the given width.
// If the renderer cannot be created or rendering fails, it returns s with surrounding whitespace trimmed.
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

// WaitForDescriptionUpdateCmd waits for a single description update from the GeminiService.
// It reads one value from gsService.DescriptionUpdates(); if the channel is closed it returns nil,
// otherwise it returns a descriptionUpdatedMsg containing the update's ID and Description.
func WaitForDescriptionUpdateCmd(gsService *gemini.GeminiService) tea.Cmd {
	return func() tea.Msg {
		update, ok := <-gsService.DescriptionUpdates()
		if !ok {
			return nil
		}

		return descriptionUpdatedMsg{ID: update.ID, Description: update.Description}
	}
}

// WaitForServiceNoticeCmd waits for a notice from the Gemini service and emits a noticeMsg containing its message.
// If the service's Notices channel is closed, it returns nil.
func WaitForServiceNoticeCmd(gsService *gemini.GeminiService) tea.Cmd {
	return func() tea.Msg {
		notice, ok := <-gsService.Notices()
		if !ok {
			return nil
		}

		return noticeMsg{text: notice.Message}
	}
}

// noticeCmd produces a command that emits a noticeMsg containing the provided text and stopLoading flag.
// The returned command, when executed, sends noticeMsg{text: text, stopLoading: stopLoading}.
func noticeCmd(text string, stopLoading bool) tea.Cmd {
	return func() tea.Msg {
		return noticeMsg{text: text, stopLoading: stopLoading}
	}
}

// summarizeUserError returns a concise user-facing message for common error conditions.
// It maps quota or rate-limit errors to "Gemini API quota exceeded. Try again shortly.",
// API key-related errors to "GOOGLE_API_KEY is missing or invalid.", timeout or deadline errors
// to "request timed out.", and otherwise returns the trimmed original error text.
func summarizeUserError(err error) string {
	if err == nil {
		return "unknown error"
	}

	msg := strings.TrimSpace(err.Error())
	lower := strings.ToLower(msg)

	switch {
	case strings.Contains(lower, "resource_exhausted"),
		strings.Contains(lower, "quota exceeded"),
		strings.Contains(lower, "rate limit"),
		strings.Contains(lower, "error 429"):
		return "Gemini API quota exceeded. Try again shortly."
	case strings.Contains(lower, "api key"):
		return "GOOGLE_API_KEY is missing or invalid."
	case strings.Contains(lower, "timeout"),
		strings.Contains(lower, "deadline exceeded"),
		strings.Contains(lower, "context deadline exceeded"):
		return "request timed out."
	default:
		return msg
	}
}
