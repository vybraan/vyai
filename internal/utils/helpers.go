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

	model := client.GenerativeModel(modelName)
	if model == nil {
		return "", fmt.Errorf("failed to get generative model")
	}

	result, err := model.GenerateContent(c, genai.Text(message))
	if err != nil {
		return "", err
	}

	return responseText(result), nil
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

func responseText(resp *genai.GenerateContentResponse) string {
	if resp == nil {
		return ""
	}

	var builder strings.Builder
	for _, cand := range resp.Candidates {
		if cand.Content == nil {
			continue
		}
		for _, part := range cand.Content.Parts {
			if text, ok := part.(genai.Text); ok {
				builder.WriteString(string(text))
			} else {
				fmt.Fprint(&builder, part)
			}
		}
	}

	return builder.String()
}
