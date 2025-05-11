package gemini

import (
	"context"
	"errors"
	"fmt"
	"strings"

	"github.com/google/generative-ai-go/genai"
)

type HistoryRepository interface {
	SendMessage(c context.Context, text genai.Text) (string, error)
	GetMessages() ([]string, error)
}

type MemoryHistoryRepository struct {
	cs *genai.ChatSession
}

func NewMemoryHistoryRepository(cs *genai.ChatSession) *MemoryHistoryRepository {
	return &MemoryHistoryRepository{
		cs: cs,
	}
}

func (mhr *MemoryHistoryRepository) GetMessages() ([]string, error) {

	if mhr.cs == nil {
		return nil, errors.New("chat session is not initialized")
	}
	if len(mhr.cs.History) == 0 {
		return nil, errors.New("no messages in history")
	}

	var messages []string
	for _, content := range mhr.cs.History {
		for _, part := range content.Parts {
			if text, ok := part.(genai.Text); ok {
				messages = append(messages, fmt.Sprintf("[Role:%s, Part:%s]", content.Role, text))
			}
		}
	}
	return messages, nil

}

func (mhr *MemoryHistoryRepository) SendMessage(c context.Context, text genai.Text) (string, error) {
	result, err := mhr.cs.SendMessage(c, text)

	if err != nil {
		return "", errors.New("failed to send a message")
	}

	var response strings.Builder

	for _, cand := range result.Candidates {
		if cand.Content != nil {
			for _, part := range cand.Content.Parts {
				fmt.Fprintln(&response, part)
			}
		}
	}
	finalResponse := response.String()
	return finalResponse, nil

}
