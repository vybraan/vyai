package gemini

import (
	"context"
	"errors"
	"fmt"
	"strings"
	"sync"

	"github.com/google/generative-ai-go/genai"
)

type Message struct {
	Role string
	Text string
}

type HistoryRepository interface {
	SendMessage(c context.Context, text genai.Text) (string, error)
	GetMessages() ([]Message, error)
}

type MemoryHistoryRepository struct {
	chatSession      *genai.ChatSession
	cachedMessages   []Message
	needsCacheUpdate bool
	messageLimit     int
	sessionFactory   func(context.Context) (*genai.ChatSession, error)
	onChange         func([]Message)
	mu               sync.RWMutex
}

func NewMemoryHistoryRepository(cs *genai.ChatSession) *MemoryHistoryRepository {
	return &MemoryHistoryRepository{
		chatSession:      cs,
		needsCacheUpdate: true,
		messageLimit:     20,
	}
}

func NewPersistentHistoryRepository(messages []Message, sessionFactory func(context.Context) (*genai.ChatSession, error), onChange func([]Message)) *MemoryHistoryRepository {
	return &MemoryHistoryRepository{
		cachedMessages:   append([]Message(nil), messages...),
		needsCacheUpdate: false,
		messageLimit:     20,
		sessionFactory:   sessionFactory,
		onChange:         onChange,
	}
}

func (mhr *MemoryHistoryRepository) GetMessages() ([]Message, error) {
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
		if len(mhr.cachedMessages) == 0 {
			return nil, errors.New("chat session is not initialized")
		}
		return append([]Message(nil), mhr.cachedMessages...), nil
	}
	if len(mhr.chatSession.History) == 0 {
		return nil, errors.New("no messages in history")
	}

	var messages []Message
	for _, content := range mhr.chatSession.History {
		for _, part := range content.Parts {
			if text, ok := part.(genai.Text); ok {
				messages = append(messages, Message{
					Role: content.Role,
					Text: string(text),
				})
			}
		}
	}
	mhr.cachedMessages = messages
	mhr.needsCacheUpdate = false
	return messages, nil
}

func (mhr *MemoryHistoryRepository) SendMessage(c context.Context, text genai.Text) (string, error) {
	if err := mhr.ensureSession(c); err != nil {
		return "", err
	}

	result, err := mhr.chatSession.SendMessage(c, text)

	if err != nil {
		return "", fmt.Errorf("send message: %w", err)
	}

	mhr.mu.Lock()
	// Prune older messages if the limit is exceeded
	if len(mhr.chatSession.History) > mhr.messageLimit {
		mhr.chatSession.History = mhr.chatSession.History[len(mhr.chatSession.History)-mhr.messageLimit:]
	}
	mhr.cachedMessages = messagesFromHistory(mhr.chatSession.History)
	mhr.needsCacheUpdate = false
	snapshot := append([]Message(nil), mhr.cachedMessages...)
	onChange := mhr.onChange
	mhr.mu.Unlock()

	if onChange != nil {
		onChange(snapshot)
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

func (mhr *MemoryHistoryRepository) ensureSession(c context.Context) error {
	mhr.mu.Lock()
	defer mhr.mu.Unlock()

	if mhr.chatSession != nil {
		return nil
	}
	if mhr.sessionFactory == nil {
		return errors.New("chat session is not initialized")
	}

	cs, err := mhr.sessionFactory(c)
	if err != nil {
		return err
	}
	cs.History = historyFromMessages(mhr.cachedMessages)
	mhr.chatSession = cs
	return nil
}

func messagesFromHistory(history []*genai.Content) []Message {
	var messages []Message
	for _, content := range history {
		for _, part := range content.Parts {
			if text, ok := part.(genai.Text); ok {
				messages = append(messages, Message{
					Role: content.Role,
					Text: string(text),
				})
			}
		}
	}
	return messages
}

func historyFromMessages(messages []Message) []*genai.Content {
	var history []*genai.Content
	for _, message := range messages {
		history = append(history, &genai.Content{
			Role:  message.Role,
			Parts: []genai.Part{genai.Text(message.Text)},
		})
	}
	return history
}
