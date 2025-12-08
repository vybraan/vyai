package gemini

import (
	"context"
	"fmt"
	"strings"

	"github.com/charmbracelet/log"
	"github.com/google/generative-ai-go/genai"
	"github.com/vybraan/vyai/internal/utils"
)

type GeminiService struct {
	cm     *ConversationManager
	logger *log.Logger
}

func NewGeminiService(cm *ConversationManager) *GeminiService {
	return &GeminiService{
		cm:     cm,
		logger: log.Default(),
	}
}

func (gs *GeminiService) SetConversationDescription(c context.Context, lock_description bool) error {
	// Only generate a description if it's the placeholder and not locked
	if gs.cm.active != nil && gs.cm.active.GetDescription() == "New Conversation..." && !gs.cm.active.IsDescriptionLocked() {
		messages, err := gs.cm.active.Repo.GetMessages()
		if err != nil {
			if err.Error() == "no messages in history" {
				return nil // No messages yet, expected during initialization
			}
			return err
		}

		conv := gs.cm.active
		go func() {
			desc, err := utils.GenerateEphemeralMessage(c, strings.Join(messages, "\n")+utils.DESCRIPTION_PROMPT)
			if err != nil {
				gs.logger.Errorf("Error generating description: %v", err)
				return
			}
			select {
			case conv.DescriptionChannel <- desc:
			case <-c.Done():
				gs.logger.Debugf("Context cancelled, not sending description")
				return
			}
		}()
		gs.cm.active.SetDescriptionLocked(lock_description)
	}
	return nil
}

func (gs *GeminiService) ClearConversation(c context.Context) error {
	conversation, err := gs.cm.GetActiveConversation()
	if err != nil {
		return err
	}

	conversation.Close()
	gs.cm.active = nil

	return nil
}

func (gs *GeminiService) NewConversation(c context.Context) (*Conversation, error) {

	// Ensure the old conversation has a description before the switch
	if err := gs.SetConversationDescription(c, true); err != nil {
		return nil, err
	}

	if gs.cm.active != nil {
		gs.cm.active.Close()
	}

	cs, err := NewChatSession(c, "gemini-2.0-flash")
	if err != nil {
		return nil, err
	}
	memRepo := NewMemoryHistoryRepository(cs)
	conversation := gs.cm.StartNewConversation(memRepo)
	return conversation, nil
}

func (gs *GeminiService) SendMessage(c context.Context, message string) (string, error) {

	conversation, err := gs.cm.GetActiveConversation()
	if err != nil {
		conversation, err = gs.NewConversation(c)
		if err != nil {
			return "", err
		}
	}

	result, err := conversation.Repo.SendMessage(c, genai.Text(message))

	if err != nil {
		return "", err
	}

	// Listen for description update
	select {
	case desc := <-conversation.DescriptionChannel:
		conversation.SetDescription(desc)
		conversation.SetDescriptionLocked(false)
	default:
	}

	// Set the first time description and set it to still be able to be updated later
	if conversation.GetDescription() == "New Conversation..." {
		gs.SetConversationDescription(c, false)
	}

	return result, nil
}

func (gs *GeminiService) GetAllConversations() ([]utils.Item, error) {
	var items []utils.Item

	gs.cm.mu.RLock()
	defer gs.cm.mu.RUnlock()

	for _, conv := range gs.cm.conversations {
		conversationItem := utils.NewItem(conv.ID, conv.GetDescription())
		items = append(items, conversationItem)
	}

	if len(items) == 0 {
		return nil, fmt.Errorf("no conversation yet")
	}

	return items, nil
}

func (gs *GeminiService) SwitchConversation(c context.Context, id string) error {

	// Ensure the old conversation has a description before the switch and then lock it
	if err := gs.SetConversationDescription(c, true); err != nil {
		return err
	}

	if gs.cm.active != nil {
		gs.cm.active.Close()
	}

	err := gs.cm.SwitchConversation(id)
	if err != nil {
		return err
	}

	return nil
}

func (gs *GeminiService) GetActiveConversation() (*Conversation, error) {
	conversation, err := gs.cm.GetActiveConversation()
	if err != nil {
		return nil, err
	}
	return conversation, nil
}
