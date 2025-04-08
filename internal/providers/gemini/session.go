package gemini

import (
	"context"
	"fmt"

	"os"

	"github.com/google/generative-ai-go/genai"
	"github.com/vybraan/vyai/internal/utils"
	"google.golang.org/api/option"
)

// NewChatSession initializes a new ChatSession with proper error handling
func NewChatSession(c context.Context, modelID string) (*genai.ChatSession, error) {
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
		Parts: []genai.Part{genai.Text(utils.SYSTEM_PROMPT)},
	}

	cs := model.StartChat()
	if cs == nil {
		return nil, fmt.Errorf("failed to start chat session")
	}

	return cs, nil
}
