package gemini

import (
	"context"
	"fmt"
	"os"
	"strings"

	"github.com/vybraan/vyai/internal/appconfig"
	"google.golang.org/genai"
)

// NewChatSession initializes a new ChatSession with proper error handling.
func NewChatSession(c context.Context, modelID string, cfg *appconfig.Config, history []Message) (*genai.Chat, error) {
	if cfg == nil {
		return nil, fmt.Errorf("config is required")
	}

	apiKey := os.Getenv("GOOGLE_API_KEY")
	if apiKey == "" {
		return nil, fmt.Errorf("GOOGLE_API_KEY environment variable is not set")
	}

	client, err := genai.NewClient(c, &genai.ClientConfig{APIKey: apiKey, Backend: genai.BackendGeminiAPI})
	if err != nil {
		return nil, fmt.Errorf("create Gemini client: %w", err)
	}

	var config *genai.GenerateContentConfig
	if strings.TrimSpace(cfg.SystemPrompt) != "" {
		config = &genai.GenerateContentConfig{
			SystemInstruction: genai.NewContentFromText(cfg.SystemPrompt, genai.RoleUser),
		}
	}

	chat, err := client.Chats.Create(c, modelID, config, historyFromMessages(history))
	if err != nil {
		return nil, fmt.Errorf("create chat session: %w", err)
	}
	return chat, nil
}
