package utils

import (
	"context"
	"fmt"
	"os"
	"strings"

	"github.com/charmbracelet/bubbles/v2/list"
	"github.com/google/generative-ai-go/genai"
	"github.com/vybraan/vyai/internal/appconfig"
	"google.golang.org/api/option"
)

func ConvertToItemList(items []Item) []list.Item {
	convertedItems := make([]list.Item, len(items))
	for i, item := range items {
		convertedItems[i] = item
	}
	return convertedItems
}

func GenerateEphemeralMessage(c context.Context, modelName string, message string) (string, error) {

	apiKey := os.Getenv("GOOGLE_API_KEY")
	if apiKey == "" {
		return "", fmt.Errorf("GOOGLE_API_KEY environment variable is not set")
	}

	client, err := genai.NewClient(c, option.WithAPIKey(apiKey))
	if err != nil {
		return "", err
	}
	defer client.Close()

	model := client.GenerativeModel(modelName)
	if model == nil {
		return "", fmt.Errorf("failed to get generative model")
	}

	result, err := model.GenerateContent(c, genai.Text(message))
	if err != nil {
		return "", err
	}

	return responseText(result)
}

func FormatSettings(cfg *appconfig.Config, apiKeySet bool) string {
	apiKeyStatus := "missing"
	if apiKeySet {
		apiKeyStatus = "set"
	}

	return strings.TrimSpace(fmt.Sprintf(`
# Settings

- App: %s
- Config dir: %s
- Config file: %s
- Data dir: %s
- Chat model: %s
- Description model: %s
- System prompt source: %s
- Description prompt source: %s
- GOOGLE_API_KEY: %s
`, cfg.AppName, cfg.ConfigDir, cfg.ConfigFile, cfg.DataDir, cfg.ChatModel, cfg.DescriptionModel, cfg.SystemPromptSource, cfg.DescriptionSource, apiKeyStatus))
}

func responseText(resp *genai.GenerateContentResponse) (string, error) {
	if resp == nil {
		return "", fmt.Errorf("empty model response")
	}
	if len(resp.Candidates) == 0 {
		return "", fmt.Errorf("model returned no candidates")
	}

	var builder strings.Builder
	candidate := resp.Candidates[0]
	if candidate.Content == nil {
		return "", fmt.Errorf("model returned empty content")
	}
	for _, part := range candidate.Content.Parts {
		if text, ok := part.(genai.Text); ok {
			builder.WriteString(string(text))
		}
	}

	text := strings.TrimSpace(builder.String())
	if text == "" {
		return "", fmt.Errorf("model returned no text content")
	}

	return text, nil
}

func SummarizeKnownError(err error) (string, bool) {
	if err == nil {
		return "", false
	}

	lower := strings.ToLower(strings.TrimSpace(err.Error()))
	switch {
	case strings.Contains(lower, "resource_exhausted"),
		strings.Contains(lower, "quota exceeded"),
		strings.Contains(lower, "rate limit"),
		strings.Contains(lower, "error 429"):
		return "Gemini API quota exceeded. Try again shortly.", true
	case strings.Contains(lower, "api key"):
		return "GOOGLE_API_KEY is missing or invalid.", true
	case strings.Contains(lower, "timeout"),
		strings.Contains(lower, "deadline exceeded"),
		strings.Contains(lower, "context deadline exceeded"):
		return "request timed out.", true
	default:
		return "", false
	}
}
