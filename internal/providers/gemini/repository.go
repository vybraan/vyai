package gemini

import (
	"context"
	"errors"
	"fmt"
	"strings"

	"github.com/google/generative-ai-go/genai"
)

type HistoryRepository interface {
	StoreMessage(chat *genai.Content) error
	SendMessage(c context.Context, text genai.Text) (string, error)
}

type MemoryHistoryRepository struct {
	cs *genai.ChatSession
}

func NewMemoryHistoryRepository(cs *genai.ChatSession) *MemoryHistoryRepository {
	return &MemoryHistoryRepository{
		cs: cs,
	}
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

func (mhr *MemoryHistoryRepository) StoreMessage(chat *genai.Content) error {

	mhr.cs.History = append(mhr.cs.History, chat)
	//this is just a mockup anyways
	// for instance i hope it will return some kind of errors
	return nil
}
