package ui

import (
	"context"
	"fmt"
	"log"
	"os"
	"os/exec"
	"regexp"
	"runtime"
	"strings"

	"github.com/charmbracelet/bubbles/v2/spinner"
	"github.com/charmbracelet/bubbles/v2/textarea"
	tea "github.com/charmbracelet/bubbletea/v2"
	"github.com/charmbracelet/glamour"
	"github.com/charmbracelet/lipgloss/v2"
	"github.com/vybraan/vyai/internal/utils"
)

func (m *UIModel) NewConversation() (*UIModel, tea.Cmd) {

	_, err := m.gsService.GetActiveConversation()
	if err != nil {
		return m, nil
	}

	err = m.gsService.ClearConversation(context.Background())
	if err != nil {

		return m, func() tea.Msg {
			return errMsg(err)
		}
		// log.Fatal(err)
		// return m, nil
	}

	m.messages = []string{}

	//clear viewport
	m.renderViewport("New conversation started")
	m.resetState()

	// Update the conversation list immediately so it doesn't bug when change to explore
	// Todo:
	// need check if needs to be in a separate routine because it might block the ui while
	//is not importand for this take of update
	items, err := m.gsService.GetAllConversations()
	if err == nil {

		m.explore.SetItems(utils.ConvertToItemList(items))
	} else {
		log.Printf("Error loading conversations: %v", err)
	}

	return m, nil

}

func (m *UIModel) openEditorForTextarea() (*UIModel, tea.Cmd) {

	temp := "vyai-conversation_*.md"
	// Open the file with $EDITOR and when done editing quiting the editot the contend of the file will be read to the variable and wtritten to the textarea
	tempFile, err := os.CreateTemp("", temp)
	if err != nil {
		log.Printf("Error creating temp file: %s", err)
		return m, nil
	}
	defer os.Remove(tempFile.Name())

	err = os.WriteFile(tempFile.Name(), []byte(m.textarea.Value()), 0644)
	if err != nil {
		log.Printf("Error writing to temp file: %s", err)
		return m, nil
	}

	editor := os.Getenv("EDITOR")

	var cmd *exec.Cmd
	if editor != "" {
		cmd = exec.Command(editor, tempFile.Name())
	} else {
		if runtime.GOOS == "windows" {
			// Not supported yet
			// cmd = exec.Command("write", tempFile.Name())
			return m, func() tea.Msg {
				return errMsg(fmt.Errorf("%s", "no editor configured. Please set $EDITOR."))
			}
		} else {
			cmd = exec.Command("vi", tempFile.Name())
		}
	}

	cmd.Stdout = os.Stdout
	cmd.Stderr = os.Stderr
	cmd.Stdin = os.Stdin

	err = cmd.Run()
	if err != nil {
		log.Printf("Error opeding the editor: %s", err)
		return m, nil
	}

	content, err := os.ReadFile(tempFile.Name())
	if err != nil {
		log.Printf("Error reading the file: %s", err)
		return m, nil
	}

	m.textarea.SetValue(string(content))
	return m, nil

}
func (m UIModel) handleKeyEnter() (UIModel, tea.Cmd) {

	switch m.activeTab {
	case 0:
		if m.state == Insert {

			prompt := m.textarea.Value()

			// renderedMarkdown := renderMarkdown(prompt)

			// m.messages = append(m.messages, m.theme.SenderStyle.Render("# [*] self:")+renderedMarkdown)
			m.messages = append(m.messages, prompt)
			m.renderViewport(strings.Join(m.messages, ""))
			m.textarea.Reset()
			m.viewport.GotoBottom()

			m.loading = true
			return m, sendMessageCmd(m, prompt)
		}
	case 1:
		i, ok := m.explore.SelectedItem().(utils.Item)
		if !ok {
			// return tea.Quit - in the future a notification
			return m, nil
		}

		m.gsService.SwitchConversation(context.Background(), i.Title())
		m.messages = []string{}

		conversation, _ := m.gsService.GetActiveConversation()

		messages, err := conversation.Repo.GetMessages()

		if err != nil {
			// No messages in the conversation
			m.renderViewport("chat is empty")
			m.resetState()

			m.activeTab = 0
			return m, nil
		}

		for _, message := range messages {
			renderedMessage := renderMarkdown(message)
			m.messages = append(m.messages, renderedMessage)
		}
		m.viewport.SetContent(
			lipgloss.NewStyle().Width(m.viewport.Width).
				Render(strings.Join(m.messages, "")))

		m.activeTab = 0
		m.resetState()

		return m, nil
	}
	return m, nil
}

// using routines here is just overkill now - Justifiable to implement cancelation in the future
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
			renderedMessage := renderMarkdown(message)
			return statusMsg(m.theme.SenderStyle.Render("# [*] vyai: ") + renderedMessage)
		case err := <-errChan:
			renderedError := renderMarkdown("# [*] System\n## Error\n * " + err.Error())
			return errMsg(fmt.Errorf("%v", renderedError))
		}
	}
}

func (m UIModel) Init() tea.Cmd {
	return tea.Batch(tea.EnterAltScreen, tea.EnableMouseAllMotion, tea.EnableMouseCellMotion, textarea.Blink)
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
	out, _ := glamour.NewTermRenderer(
		glamour.WithAutoStyle(),
		glamour.WithWordWrap(width), // defaults to 80 - need to expand
	)
	renderedMessage, _ := out.Render(strings.TrimSpace(s))

	return renderedMessage
}

func (m *UIModel) resetState() {
	m.state = Normal
	m.textarea.Reset()
	m.viewport.GotoBottom()
	m.viewport.MouseWheelEnabled = false
}
