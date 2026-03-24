package ui

import (
	"context"
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"regexp"
	"strings"

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

		m.explore.SetItems(utils.ConvertToItemList(items))
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
		i, ok := m.explore.SelectedItem().(utils.Item)
		if !ok {
			// return tea.Quit - in the future a notification
			return m, nil
		}

		if err := m.gsService.SwitchConversation(context.Background(), i.Title()); err != nil {
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

func sendAgentCmd(m UIModel, prompt string) tea.Cmd {
	return func() tea.Msg {
		userInput := strings.TrimSpace(strings.TrimPrefix(strings.TrimSpace(prompt), "/agent"))
		if userInput == "" {
			return noticeMsg{text: "Usage: /agent <promptweaver-formatted input>", stopLoading: true}
		}

		agentInput := userInput
		if !looksLikePromptWeaverInput(userInput) {
			translated, err := utils.GenerateEphemeralMessage(
				context.Background(),
				m.gsService.Config().ChatModel,
				buildAgentTranslationPrompt(userInput),
			)
			if err != nil {
				return noticeMsg{text: "Agent translation failed: " + summarizeUserError(err), stopLoading: true}
			}

			agentInput = strings.TrimSpace(translated)
			if !looksLikePromptWeaverInput(agentInput) {
				return noticeMsg{text: "Agent translation did not produce a valid tool program.", stopLoading: true}
			}
		}

		var output []string
		engine := agent.NewAgent(func(line string) {
			line = strings.TrimSpace(line)
			if line != "" {
				output = append(output, line)
			}
		}, m.workspace)

		if err := engine.Process(strings.NewReader(agentInput)); err != nil {
			return noticeMsg{text: "Agent request failed: " + summarizeUserError(err), stopLoading: true}
		}

		if len(output) == 0 {
			return noticeMsg{text: "Agent completed with no visible output.", stopLoading: true}
		}

		rendered := renderMarkdown(strings.Join(output, "\n\n"), m.width)
		return statusMsg(strings.TrimSpace(rendered))
	}
}

var promptWeaverTagPattern = regexp.MustCompile(`(?s)<\s*/?\s*([a-zA-Z][a-zA-Z0-9_-]*)`)

func looksLikePromptWeaverInput(input string) bool {
	return promptWeaverTagPattern.MatchString(input)
}

func buildAgentTranslationPrompt(userInput string) string {
	return strings.TrimSpace(fmt.Sprintf(`
You convert a user's natural-language request into PromptWeaver sections for a local coding agent.

Output rules:
- Reply with PromptWeaver tags only.
- Do not use markdown fences.
- Do not explain what you are doing outside tags.
- Use only these tags:
  - <think>hidden reasoning or short plan</think>
  - <run-bash>safe shell command</run-bash>
  - <create-file path="...">content</create-file>
  - <read-file path="..."></read-file>
  - <list-dir path="..."></list-dir>
  - <grep-file path="..." pattern="..." include="..."></grep-file>
  - <glob-file path="..." pattern="..."></glob-file>
  - <edit-file path="..." old="..." new="..."></edit-file>
  - <summary>final visible response</summary>
- Prefer read-only actions unless the user clearly asks to modify files.
- Always end with exactly one <summary>...</summary>.
- If the task is unclear or cannot be completed safely, emit only a <summary> explaining what is missing.

User request:
%s
`, userInput))
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
			return nil
		}

		return descriptionUpdatedMsg{ID: update.ID, Description: update.Description}
	}
}

func WaitForServiceNoticeCmd(gsService *gemini.GeminiService) tea.Cmd {
	return func() tea.Msg {
		notice, ok := <-gsService.Notices()
		if !ok {
			return nil
		}

		return noticeMsg{text: notice.Message}
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
