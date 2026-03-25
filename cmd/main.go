package main

import (
	"log"
	"os"

	tea "github.com/charmbracelet/bubbletea/v2"
	"github.com/vybraan/vyai/internal/agent"
	"github.com/vybraan/vyai/internal/appconfig"
	"github.com/vybraan/vyai/internal/providers/gemini"
	"github.com/vybraan/vyai/internal/ui"
	"github.com/vybraan/vyai/internal/utils"
)

// main is the program entry point. It loads application configuration, initializes the conversation manager and Gemini service (including loading stored conversations), resolves the current working directory, creates a local agent runner, and starts the Bubble Tea UI program. Initialization failures terminate the process using log.Fatal.
func main() {
	cfg, err := appconfig.Load()
	if err != nil {
		log.Fatal(err)
	}

	cm := gemini.NewConversationManager()

	gsService := gemini.NewGeminiService(cm, cfg)
	if err := gsService.LoadStoredConversations(); err != nil {
		log.Fatal(err)
	}

	workspace, err := os.Getwd()
	if err != nil {
		log.Fatal(err)
	}

	agentRunner := agent.NewLocalRunner(workspace, utils.GenerateEphemeralMessage)

	p := tea.NewProgram(ui.NewUIModel(gsService, workspace, agentRunner))
	if _, err := p.Run(); err != nil {
		log.Fatal(err)
	}
}
