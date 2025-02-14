package gemini

import (
	"context"

	"github.com/google/generative-ai-go/genai"
)

type GeminiService struct {
	hr HistoryRepository
}

func NewGeminiService(hr HistoryRepository) *GeminiService {
	return &GeminiService{
		hr: hr,
	}
}

func (gs *GeminiService) StoreMessage(role, message string) error {

	chat := &genai.Content{

		Parts: []genai.Part{
			genai.Text(message),
		},
		Role: role,
	}

	err := gs.hr.StoreMessage(chat)
	if err != nil {
		return err
	}
	return nil
}

func (gs *GeminiService) SendMessage(c context.Context, message string) (string, error) {

	result, err := gs.hr.SendMessage(c, genai.Text(message))

	if err != nil {
		return "", err
	}
	gs.StoreMessage("user", message)
	gs.StoreMessage("model", result)
	return result, nil
}
