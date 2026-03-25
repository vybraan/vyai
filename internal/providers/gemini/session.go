package gemini

import (
	"context"
	"fmt"
	"os"
	"strings"

	"github.com/google/generative-ai-go/genai"
	"github.com/vybraan/vyai/internal/appconfig"
	"google.golang.org/api/option"
)

// NewChatSession initializes a new ChatSession with proper error handling.
func NewChatSession(c context.Context, modelID string, cfg *appconfig.Config) (*genai.Client, *genai.ChatSession, error) {
	if cfg == nil {
		return nil, nil, fmt.Errorf("config is required")
	}

	apiKey := os.Getenv("GOOGLE_API_KEY")
	if apiKey == "" {
		return nil, nil, fmt.Errorf("GOOGLE_API_KEY environment variable is not set")
	}

	client, err := genai.NewClient(c, option.WithAPIKey(apiKey))
	if err != nil {
		return nil, nil, fmt.Errorf("failed to create client: %v", err)
	}

	model := client.GenerativeModel(modelID)
	if model == nil {
		_ = client.Close()
		return nil, nil, fmt.Errorf("failed to get generative model: %s", modelID)
	}

	if strings.TrimSpace(cfg.SystemPrompt) != "" {
		model.SystemInstruction = &genai.Content{
			Parts: []genai.Part{genai.Text(cfg.SystemPrompt)},
		}
	}

	cs := model.StartChat()
	if cs == nil {
		_ = client.Close()
		return nil, nil, fmt.Errorf("failed to start chat session")
	}

	return client, cs, nil
}
