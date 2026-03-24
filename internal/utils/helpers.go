package utils

import (
	"context"
	"fmt"
	"os"
	"strings"

	"github.com/charmbracelet/bubbles/v2/list"
	"github.com/google/generative-ai-go/genai"
	"google.golang.org/api/option"
)

func ConvertToItemList(items []Item) []list.Item {
	convertedItems := make([]list.Item, len(items))
	for i, item := range items {
		convertedItems[i] = item
	}
	return convertedItems
}

func GenerateEphemeralMessage(c context.Context, message string) (string, error) {

	apiKey := os.Getenv("GOOGLE_API_KEY")
	if apiKey == "" {
		return "", fmt.Errorf("GOOGLE_API_KEY environment variable is not set")
	}

	client, err := genai.NewClient(c, option.WithAPIKey(apiKey))
	if err != nil {
		return "", err
	}

	model := client.GenerativeModel("gemini-3-flash-preview")
	if model == nil {
		return "", fmt.Errorf("failed to get generative model")
	}

	result, err := model.GenerateContent(c, genai.Text(message))
	if err != nil {
		return "", err
	}

	return responseText(result), nil
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
