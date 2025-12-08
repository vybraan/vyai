package gemini

import (
	"context"
	"errors"
	"fmt"
	"strings"
	"sync"

	"github.com/google/generative-ai-go/genai"
)

type HistoryRepository interface {
	SendMessage(c context.Context, text genai.Text) (string, error)
	GetMessages() ([]string, error)
}

type MemoryHistoryRepository struct {
	chatSession      *genai.ChatSession
	cachedMessages   []string
	needsCacheUpdate bool
	messageLimit     int
	mu               sync.RWMutex
}

func NewMemoryHistoryRepository(cs *genai.ChatSession) *MemoryHistoryRepository {
	return &MemoryHistoryRepository{
		chatSession:      cs,
		needsCacheUpdate: true,
		messageLimit:     20,
	}
}

func (mhr *MemoryHistoryRepository) GetMessages() ([]string, error) {
	mhr.mu.RLock()
	if !mhr.needsCacheUpdate {
		defer mhr.mu.RUnlock()
		return mhr.cachedMessages, nil
	}
	mhr.mu.RUnlock()

	mhr.mu.Lock()
	defer mhr.mu.Unlock()

	// Re-check condition after acquiring write lock to avoid redundant work
	if !mhr.needsCacheUpdate {
		return mhr.cachedMessages, nil
	}

	if mhr.chatSession == nil {
		return nil, errors.New("chat session is not initialized")
	}
	if len(mhr.chatSession.History) == 0 {
		return nil, errors.New("no messages in history")
	}

	var messages []string
	for _, content := range mhr.chatSession.History {
		for _, part := range content.Parts {
			if text, ok := part.(genai.Text); ok {
				messages = append(messages, fmt.Sprintf("[Role:%s, Part:%s]", content.Role, text))
			}
		}
	}
	mhr.cachedMessages = messages
	mhr.needsCacheUpdate = false
	return messages, nil
}

func (mhr *MemoryHistoryRepository) SendMessage(c context.Context, text genai.Text) (string, error) {
	result, err := mhr.chatSession.SendMessage(c, text)
	mhr.needsCacheUpdate = true // Invalidate cache on new message

	// Prune older messages if the limit is exceeded
	if len(mhr.chatSession.History) > mhr.messageLimit {
		mhr.chatSession.History = mhr.chatSession.History[len(mhr.chatSession.History)-mhr.messageLimit:]
	}

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
