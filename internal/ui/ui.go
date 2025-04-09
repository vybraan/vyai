package ui

import (
	"context"
	"fmt"
	"log"
	"math/rand/v2"
	"os"
	"os/exec"
	"strconv"
	"strings"

	"github.com/charmbracelet/bubbles/list"
	"github.com/charmbracelet/bubbles/spinner"
	"github.com/charmbracelet/bubbles/textarea"
	"github.com/charmbracelet/bubbles/viewport"
	tea "github.com/charmbracelet/bubbletea"
	"github.com/charmbracelet/glamour"
	"github.com/charmbracelet/lipgloss"
	"github.com/vybraan/vyai/internal/providers/gemini"
)

const gap = "\n"

var (
	docStyle = lipgloss.NewStyle().Margin(1, 2)

	spinners = []spinner.Spinner{
		spinner.Line,
		spinner.Dot,
		spinner.MiniDot,
		spinner.Jump,
		spinner.Pulse,
		spinner.Points,
		spinner.Monkey,
	}
	spinnerStyle = lipgloss.NewStyle().Foreground(lipgloss.Color("69"))

	// ========================= Status Bar.
	//General
	statusBarStyle = lipgloss.NewStyle().
			Foreground(lipgloss.AdaptiveColor{Light: "#343433", Dark: "#C1C6B27F"}).
			Background(lipgloss.AdaptiveColor{Light: "#D9DCCF", Dark: "#353533"})

	ModeInsertStyle = lipgloss.NewStyle().
			Inherit(statusBarStyle).
			Foreground(lipgloss.Color("#000000")).
			Background(lipgloss.Color("#c3e88d")).
			Padding(0, 1)
	ModeNormalStyle = lipgloss.NewStyle().
			Inherit(statusBarStyle).
			Foreground(lipgloss.Color("#000000")).
			Background(lipgloss.Color("#7aa2f7")).
			Padding(0, 1)

	BaseNormalStyle = lipgloss.NewStyle().
			Inherit(statusBarStyle).
			Foreground(lipgloss.Color("#7aa2f7")).
			Padding(0, 1)

	BaseInsertStyle = lipgloss.NewStyle().
			Inherit(statusBarStyle).
			Foreground(lipgloss.Color("#c3e88d")).
			Padding(0, 1)

	// Top Status Bar / Navbar
	tStatusStyle = lipgloss.NewStyle().
			Inherit(statusBarStyle).
			Foreground(lipgloss.Color("#FFFDF5")).
			Background(lipgloss.Color("#ff757f")).
			Padding(0, 2).
			MarginRight(1)

	//Bottom Status Bar
	bModelTextStyle = lipgloss.NewStyle().Inherit(statusBarStyle)
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

type model struct {
	explore      list.Model
	state        State
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

func InitialModel(gs *gemini.GeminiService) model {
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
	s.Style = lipgloss.NewStyle().Foreground(lipgloss.Color("205"))

	tabs := []string{"Chat", "Explore", "Settings"}
	return model{
		state:       Normal,
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
	return tea.Batch(tea.EnterAltScreen, tea.EnableMouseAllMotion, tea.EnableMouseCellMotion, textarea.Blink, m.spinner.Tick)
}

func (m *model) resetSpinner() {

	m.spinner = spinner.New()
	m.spinner.Style = spinnerStyle
	m.spinner.Spinner = spinners[m.spinnerIndex]
}

func (m model) headerView() string {

	// Status bar
	w := lipgloss.Width
	statusKey := tStatusStyle.Render("VYAI")

	//---Tabs
	var renderedTabs []string
	for i, tab := range m.Tabs {
		if m.activeTab == i {

			if m.state == Insert {
				renderedTabs = append(renderedTabs, ModeInsertStyle.Render(tab+" (active)"))
			} else {

				renderedTabs = append(renderedTabs, ModeNormalStyle.Render(tab+" (active)"))
			}
			continue
		}
		if m.state == Insert {
			renderedTabs = append(renderedTabs, BaseInsertStyle.Render(tab))
		} else {
			renderedTabs = append(renderedTabs, BaseNormalStyle.Render(tab))
		}
	}
	tabs := lipgloss.JoinHorizontal(lipgloss.Center, renderedTabs...)

	placeholderWidth := 0
	for _, tab := range renderedTabs {
		placeholderWidth += w(tab)
	}
	placeholder := statusBarStyle.Width(m.viewport.Width - w(statusKey) - placeholderWidth).Render("")

	bar := lipgloss.JoinHorizontal(lipgloss.Top,
		statusKey,
		tabs,
		placeholder,
	)

	return bar
}

func (m model) footerView() string {

	// Status bar
	w := lipgloss.Width

	var viewPortPercent, encoding, modelKey, status string

	if m.state == Insert {
		status = ModeInsertStyle.Render(string(m.state))
		encoding = BaseInsertStyle.Render("UTF-8")
		modelKey = BaseInsertStyle.Render("Model")

		viewPortPercent = ModeInsertStyle.Render(fmt.Sprintf("%3.f%%", m.viewport.ScrollPercent()*100))
	} else {
		status = ModeNormalStyle.Render(string(m.state))
		encoding = BaseNormalStyle.Render("UTF-8")
		modelKey = BaseNormalStyle.Render("Model")

		viewPortPercent = ModeNormalStyle.Render(fmt.Sprintf("%3.f%%", m.viewport.ScrollPercent()*100))
	}

	modelVal := bModelTextStyle.
		Width(m.viewport.Width - w(modelKey) - w(status) - w(encoding) - w(viewPortPercent)).
		Render("gemini-1.5-flash")

	bar := lipgloss.JoinHorizontal(lipgloss.Top,
		status,
		modelKey,
		modelVal,
		encoding,
		viewPortPercent,
	)

	return bar
}

func (m model) Update(msg tea.Msg) (tea.Model, tea.Cmd) {
	var (
		tiCmd  tea.Cmd
		vpCmd  tea.Cmd
		expCmd tea.Cmd
		spnCmd tea.Cmd
		cmds   []tea.Cmd
	)
	// Take care of tabs now
	switch m.activeTab {
	case 0:

		if m.state == Insert {
			m.textarea, tiCmd = m.textarea.Update(msg)
		} else {
			m.viewport, vpCmd = m.viewport.Update(msg)
		}

		if !m.loading {
			spnCmd = m.spinner.Tick
		}
	case 1:

		m.explore, expCmd = m.explore.Update(msg)
	}

	cmds = append(cmds, tiCmd, vpCmd, spnCmd, expCmd)

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
			m.viewport.SetContent(lipgloss.NewStyle().Width(m.viewport.Width).Render(strings.Join(m.messages, "")))
		}
		m.viewport.GotoBottom()

		// Configs for the second Tabs

		var exp list.Model
		items, err := m.gsService.GetAllConversations()
		if err != nil {

			exp = list.New(items, list.NewDefaultDelegate(), 0, 0)
		} else {
			exp = list.New([]list.Item{}, list.NewDefaultDelegate(), 0, 0)
		}

		exp.DisableQuitKeybindings()
		exp.Title = "vyai conversation list"
		m.explore = exp

		h, v := docStyle.GetFrameSize()
		m.explore.SetSize(msg.Width-h, msg.Height-v-headerHeight)

	case tea.KeyMsg:
		switch msg.Type {
		case tea.KeyCtrlC:
			fmt.Println(m.textarea.Value())
			return m, tea.Quit
		case tea.KeyTab, tea.KeyCtrlRight, tea.KeyShiftTab, tea.KeyCtrlLeft:
			m.state = Normal
			if msg.Type == tea.KeyTab || msg.Type == tea.KeyCtrlRight {
				m.activeTab = (m.activeTab + 1) % len(m.Tabs)
			} else if msg.Type == tea.KeyShiftTab || msg.Type == tea.KeyCtrlLeft {
				m.activeTab = (m.activeTab - 1 + len(m.Tabs)) % len(m.Tabs)
			}

			if m.activeTab == 1 {
				items, err := m.gsService.GetAllConversations()
				if err == nil {
					m.explore.SetItems(items)

					m.explore.SetShowTitle(true)
				}
			}
			return m, nil
		case tea.KeyCtrlE:
			if m.activeTab == 0 && m.state == Insert {
				temp := fmt.Sprintf("vyai-conversation_*.md")
				// Open the file with $EDITOR and when done editing quiting the editot the contend of the file will be read to the variable and wtritten to the textarea
				tempFile, err := os.CreateTemp("", temp)

				if err != nil {
					log.Printf("Error: %s", err)
					return m, nil
				}

				defer os.Remove(tempFile.Name())

				editor := os.Getenv("EDITOR")

				if editor == "" {
					editor = "vi"
				}
				os.WriteFile(tempFile.Name(), []byte(m.textarea.Value()), 0644)

				cmd := exec.Command(editor, tempFile.Name())
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

				m.state = Normal
				return m, nil

			}
		case tea.KeyCtrlN:
			if m.activeTab == 0 && m.state == Normal {
				_, err := m.gsService.NewConversation(context.Background())
				if err != nil {

					log.Fatal(err)
					return m, nil
				}

				m.messages = []string{}

				//clear viewport
				m.viewport.SetContent(lipgloss.NewStyle().Width(m.viewport.Width).Render("New conversation started"))
				m.viewport.GotoBottom()
				m.viewport.MouseWheelEnabled = false
				m.state = Normal

				// Update the conversation list immediately so it doesn't bug when change to explore
				// Todo:
				// need check if needs to be in a separate routine because it might block the ui while
				//is not importand for this take of update
				items, err := m.gsService.GetAllConversations()
				if err == nil {
					m.explore.SetItems(items)
				} else {
					log.Printf("Error loading conversations: %v", err)
				}

				return m, nil

			}
		case tea.KeyEnter:
			if m.loading {
				return m, nil
			}
			prompt := m.textarea.Value()

			r, _ := glamour.NewTermRenderer(
				glamour.WithAutoStyle(),
				// glamour.WithWordWrap(40), // defaults to 80 - need to expand
			)
			out, _ := r.Render(prompt)
			m.messages = append(m.messages, m.senderStyle.Render("# [*] self:")+out)
			m.viewport.SetContent(lipgloss.NewStyle().Width(m.viewport.Width).Render(strings.Join(m.messages, "")))
			m.textarea.Reset()
			m.viewport.GotoBottom()

			m.loading = true
			return m, sendMessageCmd(m, prompt)
		default:
			switch msg.String() {
			case "i":

				m.state = Insert
				m.textarea.Focus()

				m.viewport.MouseWheelEnabled = false

				cmds = append(cmds, textarea.Blink)
			case "esc":

				m.viewport.MouseWheelEnabled = false
				m.state = Normal
				m.textarea.Blur()
			}
			if m.state == Insert {

				m.viewport.Style = lipgloss.NewStyle().BorderBottom(true).BorderStyle(lipgloss.RoundedBorder()).BorderBottomForeground(lipgloss.Color("#c3e88d"))
			} else {

				m.viewport.Style = lipgloss.NewStyle().BorderBottom(true).BorderStyle(lipgloss.RoundedBorder()).BorderBottomForeground(lipgloss.Color("#7aa2f7"))
			}

		}

	case statusMsg:
		m.messages = append(m.messages, string(msg))
		m.loading = false
		m.spinnerIndex = rand.IntN(len(spinners)-1-0) + 0
		m.resetSpinner()
		m.viewport.SetContent(lipgloss.NewStyle().Width(m.viewport.Width).Render(strings.Join(m.messages, "")))
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

func sendMessageCmd(m model, prompt string) tea.Cmd {
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
			out, _ := glamour.NewTermRenderer(glamour.WithAutoStyle())
			renderedMessage, _ := out.Render(strings.TrimSpace(message))
			return statusMsg(m.senderStyle.Render("# [*] vyai: ") + renderedMessage)
		case err := <-errChan:
			out, _ := glamour.NewTermRenderer(glamour.WithAutoStyle())
			renderedError, _ := out.Render("# [*] System\n## Error\n * " + err.Error())
			return statusMsg(renderedError)
		}
	}
}

func (m model) View() string {
	if !m.ready {
		return "\n  Initializing..."
	}

	//Tabirization
	switch m.activeTab {
	case 0:
		if m.loading {
			// Testing to show spinner when waiting for the models response
			loadGap := "\n\n"
			return fmt.Sprintf(
				"%s\n%s%s%s\n%s", m.headerView(),
				m.viewport.View(),
				loadGap,
				m.spinner.View()+" Thinking...\n", m.footerView(),
			)
		}

		return fmt.Sprintf(
			"%s\n%s%s%s\n%s", m.headerView(),
			m.viewport.View(),
			gap,
			m.textarea.View(), m.footerView(),
		)
	case 1:
		return fmt.Sprintf("%s\n%s", m.headerView(), docStyle.Render(m.explore.View()))
	default:
		return fmt.Sprintf("%s\n%s", m.headerView(), "Screwed up ailton \n * "+m.Tabs[m.activeTab]+" Page under development")

	}
}
