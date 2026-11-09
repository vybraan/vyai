package gemini

import (
	"context"
	"errors"
	"fmt"
	"strings"
	"sync"

	"google.golang.org/genai"
)

type Message struct {
	Role string
	Text string
}

var (
	ErrSessionNotInitialized = errors.New("chat session is not initialized")
	ErrNoMessagesInHistory   = errors.New("no messages in history")
)

type HistoryRepository interface {
	SendMessage(c context.Context, text string) (string, error)
	SendMessageStream(c context.Context, text string, onToken func(string)) (string, error)
	GetMessages() ([]Message, error)
	ResetSession()
}

type MemoryHistoryRepository struct {
	chatSession    *genai.Chat
	cachedMessages []Message
	messageLimit   int
	sessionFactory func(context.Context, []Message) (*genai.Chat, error)
	onChange       func([]Message)
	mu             sync.RWMutex
}

func NewMemoryHistoryRepository(cs *genai.Chat) *MemoryHistoryRepository {
	var messages []Message
	if cs != nil {
		messages = messagesFromHistory(cs.History(true))
	}
	return &MemoryHistoryRepository{
		chatSession:    cs,
		cachedMessages: messages,
		messageLimit:   20,
	}
}

func NewPersistentHistoryRepository(messages []Message, sessionFactory func(context.Context, []Message) (*genai.Chat, error), onChange func([]Message)) *MemoryHistoryRepository {
	return &MemoryHistoryRepository{
		cachedMessages: append([]Message(nil), messages...),
		messageLimit:   20,
		sessionFactory: sessionFactory,
		onChange:       onChange,
	}
}

func (mhr *MemoryHistoryRepository) GetMessages() ([]Message, error) {
	mhr.mu.RLock()
	defer mhr.mu.RUnlock()
	if len(mhr.cachedMessages) == 0 {
		return nil, ErrNoMessagesInHistory
	}
	return append([]Message(nil), mhr.cachedMessages...), nil
}

func (mhr *MemoryHistoryRepository) SendMessage(c context.Context, text string) (string, error) {
	if err := mhr.ensureSession(c); err != nil {
		return "", err
	}

	result, err := mhr.chatSession.SendMessage(c, genai.Part{Text: text})

	if err != nil {
		return "", fmt.Errorf("send message: %w", err)
	}

	if result == nil {
		return "", fmt.Errorf("send message: nil response")
	}
	mhr.captureHistory()
	return result.Text(), nil
}

func (mhr *MemoryHistoryRepository) SendMessageStream(c context.Context, text string, onToken func(string)) (string, error) {
	if err := mhr.ensureSession(c); err != nil {
		return "", err
	}

	var fullResponse strings.Builder
	for resp, err := range mhr.chatSession.SendMessageStream(c, genai.Part{Text: text}) {
		if err != nil {
			return "", fmt.Errorf("send message: %w", err)
		}
		if resp == nil {
			continue
		}
		chunk := resp.Text()
		if chunk != "" {
			fullResponse.WriteString(chunk)
			onToken(chunk)
		}
	}
	mhr.captureHistory()
	return fullResponse.String(), nil
}

func (mhr *MemoryHistoryRepository) captureHistory() {
	mhr.mu.Lock()
	mhr.cachedMessages = messagesFromHistory(mhr.chatSession.History(true))
	if len(mhr.cachedMessages) > mhr.messageLimit {
		mhr.cachedMessages = mhr.cachedMessages[len(mhr.cachedMessages)-mhr.messageLimit:]
		mhr.chatSession = nil
	}
	snapshot := append([]Message(nil), mhr.cachedMessages...)
	onChange := mhr.onChange
	mhr.mu.Unlock()

	if onChange != nil {
		onChange(snapshot)
	}

}

func (mhr *MemoryHistoryRepository) ensureSession(c context.Context) error {
	mhr.mu.Lock()
	defer mhr.mu.Unlock()

	if mhr.chatSession != nil {
		return nil
	}
	if mhr.sessionFactory == nil {
		return ErrSessionNotInitialized
	}

	cs, err := mhr.sessionFactory(c, append([]Message(nil), mhr.cachedMessages...))
	if err != nil {
		return err
	}
	mhr.chatSession = cs
	return nil
}

func messagesFromHistory(history []*genai.Content) []Message {
	var messages []Message
	for _, content := range history {
		for _, part := range content.Parts {
			if part != nil && part.Text != "" && !part.Thought {
				messages = append(messages, Message{
					Role: content.Role,
					Text: part.Text,
				})
			}
		}
	}
	return messages
}

func historyFromMessages(messages []Message) []*genai.Content {
	var history []*genai.Content
	for _, message := range messages {
		history = append(history, genai.NewContentFromText(message.Text, genai.Role(message.Role)))
	}
	return history
}

func (mhr *MemoryHistoryRepository) ResetSession() {
	mhr.mu.Lock()
	defer mhr.mu.Unlock()

	mhr.chatSession = nil
}
