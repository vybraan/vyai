package gemini

import (
	"context"
	"fmt"

	"github.com/charmbracelet/bubbles/list"
	"github.com/google/generative-ai-go/genai"
	"github.com/vybraan/vyai/internal/utils"
)

type item struct {
	title string
	desc  string
}

func NewItem(title, description string) *item {
	return &item{
		title: title,
		desc:  description,
	}
}

func (i item) Title() string       { return i.title }
func (i item) Description() string { return i.desc }
func (i item) FilterValue() string { return i.desc }

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
	if gs.cm.active != nil && gs.cm.active.DescriptionLocked != true {
		desc, err := gs.SendEphemeralMessage(c, utils.DESCRIPTION_PROMPT)
		if err != nil {
			return err
		}
		gs.cm.active.SetDescription(desc)
		gs.cm.active.DescriptionLocked = lock_description
	}
	return nil

}

func (gs *GeminiService) NewConversation(c context.Context) (*Conversation, error) {

	// Ensure the conversation has a description before the switch
	if err := gs.SetConversationDescription(c, true); err != nil {
		return nil, err
	}

	cs, err := NewChatSession(c, "gemini-1.5-flash")
	if err != nil {
		return nil, err
	}
	memRepo := NewMemoryHistoryRepository(cs)
	convo := gs.cm.StartNewConversation(memRepo)
	return convo, nil
}

func (gs *GeminiService) SendMessage(c context.Context, message string) (string, error) {

	convo, err := gs.cm.GetActiveConversation()
	if err != nil {
		convo, err = gs.NewConversation(c)
		if err != nil {
			return "", err
		}
	}

	result, err := convo.Repo.SendMessage(c, genai.Text(message))

	if err != nil {
		return "", err
	}

	//Set the first time description and set it to still be able to be updated later
	if convo.Description == "" {
		gs.SetConversationDescription(c, false)
	}

	return result, nil
}

func (gs *GeminiService) SendEphemeralMessage(c context.Context, message string) (string, error) {

	convo, err := gs.cm.GetActiveConversation()
	if err != nil {

		convo, err = gs.NewConversation(c)
		return "", err
	}

	result, err := convo.Repo.SendMessage(c, genai.Text(message))

	if err != nil {
		return "", err
	}
	return result, nil
}

func (gs *GeminiService) GetAllConversations() ([]list.Item, error) {
	var items []list.Item

	for _, conv := range gs.cm.conversations {
		convoItem := item{
			title: conv.ID,
			desc:  conv.Description,
		}
		items = append(items, convoItem)
	}

	if len(items) == 0 {
		return nil, fmt.Errorf("no conversation yet")
	}

	return items, nil
}
