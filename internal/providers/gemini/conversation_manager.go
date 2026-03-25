package gemini

import (
	"fmt"
	"sort"
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
	return cm.StartNewConversationWithModel(repo, "")
}

func (cm *ConversationManager) StartNewConversationWithModel(repo HistoryRepository, chatModel string) *Conversation {
	cm.mu.Lock()
	defer cm.mu.Unlock()

	conversation := NewConversation(repo, chatModel)
	cm.conversations[conversation.ID] = conversation

	cm.active = conversation
	return conversation
}

func (cm *ConversationManager) AddConversation(conversation *Conversation) {
	cm.mu.Lock()
	defer cm.mu.Unlock()

	cm.conversations[conversation.ID] = conversation
}

func (cm *ConversationManager) SwitchConversation(id string) error {
	cm.mu.Lock()
	defer cm.mu.Unlock()

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
	return conversation.GetDescription(), nil
}

func (cm *ConversationManager) RemoveConversation(id string) (*Conversation, error) {
	cm.mu.Lock()
	defer cm.mu.Unlock()

	conversation, exists := cm.conversations[id]
	if !exists {
		return nil, fmt.Errorf("conversation with ID %s does not exist", id)
	}

	delete(cm.conversations, id)
	if cm.active != nil && cm.active.ID == id {
		cm.active = nil
	}

	return conversation, nil
}

func (cm *ConversationManager) All() []*Conversation {
	cm.mu.RLock()
	defer cm.mu.RUnlock()

	conversations := make([]*Conversation, 0, len(cm.conversations))
	for _, conv := range cm.conversations {
		conversations = append(conversations, conv)
	}

	sort.Slice(conversations, func(i, j int) bool {
		return conversations[i].UpdatedAt.After(conversations[j].UpdatedAt)
	})

	return conversations
}
