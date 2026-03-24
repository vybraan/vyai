package main

import (
	"log"

	tea "github.com/charmbracelet/bubbletea/v2"
	"github.com/vybraan/vyai/internal/appconfig"
	"github.com/vybraan/vyai/internal/providers/gemini"
	"github.com/vybraan/vyai/internal/ui"
)

func main() {
	cfg, err := appconfig.Load()
	if err != nil {
		log.Fatal(err)
	}

	cm := gemini.NewConversationManager()

	gsService := gemini.NewGeminiService(cm, cfg)

	p := tea.NewProgram(ui.NewUIModel(gsService))
	if _, err := p.Run(); err != nil {
		log.Fatal(err)
	}
}
