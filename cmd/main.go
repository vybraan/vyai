package main

import (
	"context"
	"fmt"
	"log"
	"math/rand/v2"
	"os"
	"strings"

	"github.com/charmbracelet/bubbles/spinner"
	"github.com/charmbracelet/bubbles/textarea"
	"github.com/charmbracelet/bubbles/viewport"
	tea "github.com/charmbracelet/bubbletea"
	"github.com/charmbracelet/lipgloss"
	"github.com/google/generative-ai-go/genai"
	"github.com/vybraan/vyai/internal/providers/gemini"
	"google.golang.org/api/option"
)

const gap = "\n"

const SYSTEM_PROMPT = `
	    You are a Linux System Admin Assistant. Your role is to help users become proficient in Linux and infrastructure management. Provide clear, concise, and direct answers to technical questions. Avoid verbosity and focus on actionable guidance. Assist with:
	
	    Linux commands and scripting
	    System administration tasks
	    Networking and security best practices
	    Programming concepts and languages
	    Troubleshooting and problem-solving
	
	If the user greets you, respond with a friendly greeting and share a fun fact about Linux, Unix, or a programming tip. Always prioritize clarity and brevity in your responses.
	    `

func main() {
	c := context.Background()
	client, err := genai.NewClient(c, option.WithAPIKey(os.Getenv("GOOGLE_API_KEY")))
	if err != nil {
		log.Fatal(err)
	}
	defer client.Close()

	model := client.GenerativeModel("gemini-1.5-flash")
	model.SystemInstruction = &genai.Content{
		Parts: []genai.Part{genai.Text(SYSTEM_PROMPT)},
	}
	cs := model.StartChat()

	memRepo := gemini.NewMemoryHistoryRepository(cs)
	gsService := gemini.NewGeminiService(memRepo)

	p := tea.NewProgram(initialModel(gsService), tea.WithAltScreen(), tea.WithMouseCellMotion())

	if _, err := p.Run(); err != nil {
		log.Fatal(err)
	}
}

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

	textStyle    = lipgloss.NewStyle().Foreground(lipgloss.Color("252")).Render
	spinnerStyle = lipgloss.NewStyle().Foreground(lipgloss.Color("69"))
	helpStyle    = lipgloss.NewStyle().Foreground(lipgloss.Color("241")).Render
	docStyle     = lipgloss.NewStyle().Padding(1, 2, 1, 2)
	titleStyle   = func() lipgloss.Style {
		b := lipgloss.RoundedBorder()
		b.Right = "├"
		return lipgloss.NewStyle().BorderStyle(b).Padding(0, 1)
	}()

	infoStyle = func() lipgloss.Style {
		b := lipgloss.RoundedBorder()
		b.Left = "┤"
		return titleStyle.BorderStyle(b)
	}()
)

type (
	errMsg    error
	statusMsg string
)

type model struct {
	viewport     viewport.Model
	messages     []string
	textarea     textarea.Model
	senderStyle  lipgloss.Style
	gsService    *gemini.GeminiService
	err          error
	spinner      spinner.Model
	spinnerIndex int
	loading      bool
	Tabs         []string
	activeTab    int
	ready        bool
}

func initialModel(gs *gemini.GeminiService) model {
	ta := textarea.New()
	ta.Placeholder = "Send a message..."
	ta.Focus()

	ta.Prompt = "┃ "
	ta.CharLimit = 280

	ta.SetWidth(30)
	ta.SetHeight(3)

	ta.FocusedStyle.CursorLine = lipgloss.NewStyle()
	ta.ShowLineNumbers = false
	ta.KeyMap.InsertNewline.SetEnabled(false)

	s := spinner.New()
	s.Spinner = spinner.Dot
	s.Style = lipgloss.NewStyle().Foreground(lipgloss.Color("205"))

	tabs := []string{"Chat", "Explore"}

	return model{
		gsService:   gs,
		textarea:    ta,
		messages:    []string{},
		senderStyle: lipgloss.NewStyle().Foreground(lipgloss.Color("5")),
		err:         nil,
		spinner:     s,
		loading:     false,
		Tabs:        tabs,
		activeTab:   0,
	}
}

func (m model) Init() tea.Cmd {
	return tea.Batch(tea.EnterAltScreen, textarea.Blink, m.spinner.Tick)
}

func (m *model) resetSpinner() {

	m.spinner = spinner.New()
	m.spinner.Style = spinnerStyle
	m.spinner.Spinner = spinners[m.spinnerIndex]
}

func (m model) headerView() string {

	var renderedTabs []string
	for _, tab := range m.Tabs {
		renderedTabs = append(renderedTabs, titleStyle.Render(tab))
	}

	row := lipgloss.JoinHorizontal(lipgloss.Center, m.Tabs...)

	line := strings.Repeat("─", max(0, m.viewport.Width-lipgloss.Width(row)))
	return lipgloss.JoinHorizontal(lipgloss.Center, renderedTabs[0], renderedTabs[1], line)
}

func (m model) footerView() string {
	info := infoStyle.Render(fmt.Sprintf("%3.f%%", m.viewport.ScrollPercent()*100))
	line := strings.Repeat("─", max(0, m.viewport.Width-lipgloss.Width(info)))
	return lipgloss.JoinHorizontal(lipgloss.Center, line, info)
}
func (m model) Update(msg tea.Msg) (tea.Model, tea.Cmd) {
	var (
		tiCmd tea.Cmd
		vpCmd tea.Cmd
		cmds  []tea.Cmd
	)

	m.textarea, tiCmd = m.textarea.Update(msg)
	m.viewport, vpCmd = m.viewport.Update(msg)
	cmds = append(cmds, tiCmd, vpCmd, m.spinner.Tick)

	switch msg := msg.(type) {
	case tea.WindowSizeMsg:
		headerHeight := lipgloss.Height(m.headerView())
		footerHeight := lipgloss.Height(m.footerView())
		textareaHeight := lipgloss.Height(string(m.textarea.Height())) // temoprary, probably should find a better way of doing this
		verticalMarginHeight := headerHeight + footerHeight + textareaHeight + textareaHeight + textareaHeight

		if !m.ready {

			// vp := viewport.New(30, 5)
			m.viewport = viewport.New(msg.Width, msg.Height-verticalMarginHeight)
			m.viewport.SetContent(`Welcome to vyai - cli interface for AI!`)
			// m.viewport.YPosition = headerHeight

			m.ready = true
		} else {
			m.viewport.Width = msg.Width
			m.viewport.Height = msg.Height - verticalMarginHeight

			// m.viewport.Height = msg.Height - m.textarea.Height() - lipgloss.Height(gap)
			// m.viewport.Height = msg.Height - m.textarea.Height() - lipgloss.Height(gap) - verticalMarginHeight

		}

		m.textarea.SetWidth(msg.Width)
		if len(m.messages) > 0 {
			// Wrap content before setting it.
			m.viewport.SetContent(lipgloss.NewStyle().Width(m.viewport.Width).Render(strings.Join(m.messages, "")))
		}
		m.viewport.GotoBottom()
	case tea.KeyMsg:
		switch msg.Type {
		case tea.KeyCtrlC, tea.KeyEsc:
			fmt.Println(m.textarea.Value())
			return m, tea.Quit
		case tea.KeyTab, tea.KeyCtrlRight:
			m.activeTab = (m.activeTab + 1) % len(m.Tabs)
			return m, nil
		case tea.KeyShiftTab, tea.KeyCtrlLeft:
			m.activeTab = (m.activeTab - 1 + len(m.Tabs)) % len(m.Tabs)
			return m, nil
		case tea.KeyEnter:
			if m.loading {
				return m, nil
			}
			prompt := m.textarea.Value()
			m.messages = append(m.messages, m.senderStyle.Render("Me: ")+m.textarea.Value())
			m.viewport.SetContent(lipgloss.NewStyle().Width(m.viewport.Width).Render(strings.Join(m.messages, "\n")))
			m.textarea.Reset()
			m.viewport.GotoBottom()

			m.loading = true
			return m, sendMessageCmd(m, prompt)
		}

	case statusMsg:
		m.messages = append(m.messages, m.senderStyle.Render("")+string(msg))
		m.loading = false
		m.spinnerIndex = rand.IntN(len(spinners)-1-0) + 0
		m.resetSpinner()
		m.viewport.SetContent(lipgloss.NewStyle().Width(m.viewport.Width).Render(strings.Join(m.messages, "\n")))
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
		m.viewport.SetContent(lipgloss.NewStyle().Width(m.viewport.Width).Render(strings.Join(m.messages, "\n")))
		m.viewport.GotoBottom()
		return m, nil
	}
	return m, tea.Batch(cmds...)
}

// tea.Cmd to receive the modells message without blocking UI
func sendMessageCmd(m model, prompt string) tea.Cmd {
	return func() tea.Msg {
		message, err := m.gsService.SendMessage(context.Background(), prompt)
		if err != nil {
			return statusMsg("System: There was an error processing your message")
		}
		return statusMsg(m.senderStyle.Render("vyai: ") + strings.TrimSpace(message))
	}
}
func (m model) View() string {
	if !m.ready {
		return "\n  Initializing..."
	}

	if m.activeTab == 0 {
		if m.loading {
			// Testing to show spinner when waiting for the models response
			return fmt.Sprintf(
				"%s\n%s%s%s\n%s", m.headerView(),
				m.viewport.View(),
				gap, m.footerView(),
				m.spinner.View()+" Thinking...",
			)
		}

		return fmt.Sprintf(
			"%s\n%s%s%s\n%s", m.headerView(),
			m.viewport.View(),
			gap, m.footerView(),
			m.textarea.View(),
		)
	} else {
		return fmt.Sprintf("%s%s", m.headerView(), "Screw it ailton")
	}
}
