package utils

import (
	"context"
	"fmt"
	"os"

	"github.com/charmbracelet/bubbles/list"

	"google.golang.org/genai"
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

	client, err := genai.NewClient(c, &genai.ClientConfig{
		APIKey:  apiKey,
		Backend: genai.BackendGeminiAPI,
	})
	if err != nil {
		return "", err
	}
	thinkingBudgetVal := int32(0)

	result, err := client.Models.GenerateContent(
		c,
		"gemini-2.0-flash",
		genai.Text(message),
		&genai.GenerateContentConfig{
			ThinkingConfig: &genai.ThinkingConfig{
				ThinkingBudget: &thinkingBudgetVal, // Disables thinking
			},
		},
	)
	if err != nil {
		return "", err
	}

	return result.Text(), nil
}
