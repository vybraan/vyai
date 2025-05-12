package gemini

import (
	"context"
	"fmt"

	"github.com/google/generative-ai-go/genai"
	"github.com/vybraan/vyai/internal/utils"
)

type GeminiService struct {
	cm *ConversationManager
}

func NewGeminiService(cm *ConversationManager) *GeminiService {
	return &GeminiService{
		cm: cm,
	}
}

func (gs *GeminiService) SetConversationDescription(c context.Context, lock_description bool) error {

	// use send epheral messages to ask the model to give a description to the chat
	// and store it in the conversation
	// this is a workaround for the fact that the  does not have a Description
	// field in the conversation when newly created so when create a new one before
	// it happens give it description
	if gs.cm.active != nil && !gs.cm.active.DescriptionLocked {
		desc, err := gs.SendEphemeralMessage(c, utils.DESCRIPTION_PROMPT)
		if err != nil {
			return err
		}
		gs.cm.active.SetDescription(desc)
		gs.cm.active.DescriptionLocked = lock_description
	}
	return nil

}

func (gs *GeminiService) ClearConversation(c context.Context) error {
	_, err := gs.cm.GetActiveConversation()
	if err != nil {
		return err
	}

	gs.cm.active = nil

	return nil
}

func (gs *GeminiService) NewConversation(c context.Context) (*Conversation, error) {

	// Ensure the old conversation has a description before the switch
	if err := gs.SetConversationDescription(c, true); err != nil {
		return nil, err
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

	//Set the first time description and set it to still be able to be updated later
	if conversation.Description == "" {
		gs.SetConversationDescription(c, false)
	}

	return result, nil
}

func (gs *GeminiService) SendEphemeralMessage(c context.Context, message string) (string, error) {

	conversation, err := gs.cm.GetActiveConversation()
	if err != nil {

		_, err = gs.NewConversation(c)
		return "", err
	}

	result, err := conversation.Repo.SendMessage(c, genai.Text(message))

	if err != nil {
		return "", err
	}
	return result, nil
}

func (gs *GeminiService) GetAllConversations() ([]utils.Item, error) {
	var items []utils.Item

	for _, conv := range gs.cm.conversations {
		conversationItem := utils.NewItem(conv.ID, conv.Description)
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

	err := gs.cm.SwitchConversation(id)
	if err != nil {
		return err
	}

	// if gs.cm.active != nil {
	// 	gs.cm.active.DescriptionLocked = true
	// }

	return nil
}

func (gs *GeminiService) GetActiveConversation() (*Conversation, error) {
	conversation, err := gs.cm.GetActiveConversation()
	if err != nil {
		return nil, err
	}
	return conversation, nil
}
