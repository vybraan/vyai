package gemini

import (
	"context"
	"fmt"

	"os"

	"github.com/google/generative-ai-go/genai"
	"github.com/vybraan/vyai/internal/appconfig"
	"google.golang.org/api/option"
)

// NewChatSession creates a new genai.ChatSession for the given model ID and config.
// It initializes the model's system instruction from cfg.SystemPrompt and returns the started chat session.
// The function returns an error if the GOOGLE_API_KEY environment variable is missing, if the Gemini client
// cannot be created, if the specified generative model cannot be obtained, or if a chat session cannot be started.
func NewChatSession(c context.Context, modelID string, cfg *appconfig.Config) (*genai.ChatSession, error) {
	apiKey := os.Getenv("GOOGLE_API_KEY")
	if apiKey == "" {
		return nil, fmt.Errorf("GOOGLE_API_KEY environment variable is not set")
	}

	client, err := genai.NewClient(c, option.WithAPIKey(apiKey))
	if err != nil {
		return nil, fmt.Errorf("failed to create client: %v", err)
	}

	model := client.GenerativeModel(modelID)
	if model == nil {
		return nil, fmt.Errorf("failed to get generative model: %s", modelID)
	}

	model.SystemInstruction = &genai.Content{
		Parts: []genai.Part{genai.Text(cfg.SystemPrompt)},
	}

	cs := model.StartChat()
	if cs == nil {
		return nil, fmt.Errorf("failed to start chat session")
	}

	return cs, nil
}
