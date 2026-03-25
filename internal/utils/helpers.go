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

// ConvertToItemList converts a slice of Item to a slice of list.Item.
// It returns a new slice with the same length where each element is assigned from the input slice.
func ConvertToItemList(items []Item) []list.Item {
	convertedItems := make([]list.Item, len(items))
	for i, item := range items {
		convertedItems[i] = item
	}
	return convertedItems
}

// GenerateEphemeralMessage produces a short generated message from the specified model using the provided input text.
// 
// It returns the generated text or an error. Errors occur if the GOOGLE_API_KEY environment variable is not set,
// if the generative AI client cannot be created, if the requested model cannot be obtained, or if content generation fails.
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

// FormatSettings formats a trimmed, multi-line Markdown summary of the provided configuration and Google API key status.
// 
// The apiKeySet flag indicates whether GOOGLE_API_KEY is present and is rendered as "set" or "missing" in the output.
// The returned string contains a heading and bullet lines for App, Config dir, Config file, Data dir, Chat model, Description model,
// System prompt source, Description prompt source, and the derived GOOGLE_API_KEY status.
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

// responseText extracts and concatenates textual content from a GenerateContentResponse.
// If resp is nil, it returns an empty string. It skips candidates with nil Content and, for each
// content part, appends the part's string value when the part is a genai.Text; otherwise it formats
// the part and appends the formatted representation.
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
