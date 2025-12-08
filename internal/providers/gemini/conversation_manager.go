package gemini

import (
	"fmt"
	"sync"
)

type ConversationManager struct {
	mu            sync.RWMutex
	conversations map[string]*Conversation
	active        *Conversation
}

func NewConversationManager() *ConversationManager {
	return &ConversationManager{
		conversations: make(map[string]*Conversation),
	}
}

func (cm *ConversationManager) StartNewConversation(repo HistoryRepository) *Conversation {
	cm.mu.Lock()
	defer cm.mu.Unlock()

	conversation := NewConversation(repo)
	cm.conversations[conversation.ID] = conversation

	cm.active = conversation
	return conversation
}

func (cm *ConversationManager) SwitchConversation(id string) error {
	cm.mu.RLock()
	defer cm.mu.RUnlock()

	conversation, exists := cm.conversations[id]
	if !exists {
		return fmt.Errorf("conversation with ID %s does not exist", id)
	}

	cm.active = conversation
	return nil
}

func (cm *ConversationManager) GetActiveConversation() (*Conversation, error) {
	cm.mu.RLock()
	defer cm.mu.RUnlock()

	if cm.active == nil {
		return nil, fmt.Errorf("no active conversation found")
	}
	return cm.active, nil
}

func (cm *ConversationManager) GetConversationDescription(id string) (string, error) {
	cm.mu.RLock()
	defer cm.mu.RUnlock()

	conversation, exists := cm.conversations[id]
	if !exists {
		return "", fmt.Errorf("no description found for conversation %s", id)
	}
	return conversation.description, nil
}
