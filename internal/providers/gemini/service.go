package gemini

import (
	"context"
	"fmt"
	"strings"

	"github.com/charmbracelet/log"
	"github.com/google/generative-ai-go/genai"
	"github.com/vybraan/vyai/internal/utils"
)

type DescriptionUpdate struct {
	ID          string
	Description string
}

type Notice struct {
	Message string
}

type GeminiService struct {
	cm                 *ConversationManager
	logger             *log.Logger
	descriptionUpdates chan DescriptionUpdate
	notices            chan Notice
}

func NewGeminiService(cm *ConversationManager) *GeminiService {
	return &GeminiService{
		cm:                 cm,
		logger:             log.Default(),
		descriptionUpdates: make(chan DescriptionUpdate, 8),
		notices:            make(chan Notice, 8),
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
			defer func() {
				if r := recover(); r != nil {
					gs.logger.Errorf("Recovered from panic in SetConversationDescription goroutine: %v", r)
				}
			}()
			desc, err := utils.GenerateEphemeralMessage(c, buildDescriptionPrompt(messages)+utils.DESCRIPTION_PROMPT)
			if err != nil {
				notice := summarizeGeminiError("Conversation title was not updated", err)
				gs.publishNotice(notice)
				gs.logger.Warnf("%s", notice)
				return
			}
			desc = strings.TrimSpace(desc)
			if desc == "" {
				return
			}

			conv.SetDescription(desc)

			select {
			case gs.descriptionUpdates <- DescriptionUpdate{ID: conv.ID, Description: desc}:
			case <-c.Done():
				gs.logger.Debugf("Context cancelled, not publishing description update")
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
	gs.cm.mu.Lock()
	gs.cm.active = nil
	gs.cm.mu.Unlock()

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

	cs, err := NewChatSession(c, "gemini-3-flash-preview")
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

func (gs *GeminiService) DescriptionUpdates() <-chan DescriptionUpdate {
	return gs.descriptionUpdates
}

func (gs *GeminiService) Notices() <-chan Notice {
	return gs.notices
}

func buildDescriptionPrompt(messages []Message) string {
	var parts []string
	for _, message := range messages {
		parts = append(parts, fmt.Sprintf("[%s] %s", message.Role, message.Text))
	}

	return strings.Join(parts, "\n")
}

func (gs *GeminiService) publishNotice(message string) {
	message = strings.TrimSpace(message)
	if message == "" {
		return
	}

	select {
	case gs.notices <- Notice{Message: message}:
	default:
	}
}

func summarizeGeminiError(prefix string, err error) string {
	if err == nil {
		return prefix
	}

	raw := err.Error()
	lower := strings.ToLower(raw)

	switch {
	case strings.Contains(lower, "resource_exhausted"),
		strings.Contains(lower, "quota exceeded"),
		strings.Contains(lower, "rate limit"),
		strings.Contains(lower, "error 429"):
		return prefix + ": Gemini API quota exceeded. Try again shortly."
	case strings.Contains(lower, "api key"):
		return prefix + ": GOOGLE_API_KEY is missing or invalid."
	case strings.Contains(lower, "deadline exceeded"),
		strings.Contains(lower, "context deadline exceeded"),
		strings.Contains(lower, "timeout"):
		return prefix + ": request timed out."
	default:
		return prefix + ": request failed."
	}
}
