package gemini

import (
	"context"
	"errors"
	"fmt"
	"os"
	"strings"

	"github.com/charmbracelet/log"
	"github.com/google/generative-ai-go/genai"
	"github.com/vybraan/vyai/internal/appconfig"
	"github.com/vybraan/vyai/internal/utils"
)

type DescriptionUpdate struct {
	ID          string
	Description string
}

type Notice struct {
	Message string
}

type ConversationSummary struct {
	ID          string
	Description string
	ChatModel   string
	UpdatedAt   string
}

type GeminiService struct {
	cm                 *ConversationManager
	cfg                *appconfig.Config
	store              *FileConversationStore
	logger             *log.Logger
	descriptionUpdates chan DescriptionUpdate
	notices            chan Notice
}

func NewGeminiService(cm *ConversationManager, cfg *appconfig.Config) *GeminiService {
	return &GeminiService{
		cm:                 cm,
		cfg:                cfg,
		store:              NewFileConversationStore(cfg.DataDir),
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
			if errors.Is(err, ErrNoMessagesInHistory) {
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
			desc, err := utils.GenerateEphemeralMessage(c, gs.cfg.DescriptionModel, buildDescriptionPrompt(messages)+gs.cfg.DescriptionPrompt)
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
			gs.persistConversation(conv)

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

	conversation := gs.cm.StartNewConversationWithModel(nil, gs.cfg.ChatModel)
	memRepo := NewPersistentHistoryRepository(nil, func(ctx context.Context) (interface{ Close() error }, *genai.ChatSession, error) {
		return NewChatSession(ctx, gs.cfg.ChatModel, gs.cfg)
	}, func(_ []Message) {
		conversation.Touch()
		gs.persistConversation(conversation)
	})
	conversation.Repo = memRepo
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
	conversation.Touch()

	// Set the first time description and set it to still be able to be updated later
	if conversation.GetDescription() == "New Conversation..." {
		gs.SetConversationDescription(c, false)
	}

	return result, nil
}

func (gs *GeminiService) GetAllConversations() ([]ConversationSummary, error) {
	conversations := gs.cm.All()
	if len(conversations) == 0 {
		return nil, fmt.Errorf("no conversation yet")
	}

	summaries := make([]ConversationSummary, 0, len(conversations))
	for _, conv := range conversations {
		summaries = append(summaries, ConversationSummary{
			ID:          conv.ID,
			Description: conv.GetDescription(),
			ChatModel:   conv.ChatModel,
			UpdatedAt:   conv.UpdatedAtSnapshot().Format("2006-01-02 15:04"),
		})
	}

	return summaries, nil
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

func (gs *GeminiService) Config() *appconfig.Config {
	return gs.cfg
}

func (gs *GeminiService) SettingsMarkdown() string {
	return utils.FormatSettings(gs.cfg, os.Getenv("GOOGLE_API_KEY") != "")
}

func (gs *GeminiService) ReloadConfig() error {
	oldCfg := gs.cfg

	cfg, err := appconfig.Load()
	if err != nil {
		return err
	}

	gs.cfg = cfg
	gs.store = NewFileConversationStore(cfg.DataDir)
	for _, conv := range gs.cm.All() {
		if conv.ChatModel == oldCfg.ChatModel {
			conv.ChatModel = cfg.ChatModel
		}
		conv.Repo.ResetSession()
		gs.persistConversation(conv)
	}
	return nil
}

func (gs *GeminiService) LoadStoredConversations() error {
	records, err := gs.store.LoadAll()
	if err != nil {
		return err
	}

	for _, record := range records {
		record := record
		if record.ChatModel == "" {
			record.ChatModel = gs.cfg.ChatModel
		}
		repo := NewPersistentHistoryRepository(record.Messages, func(ctx context.Context) (interface{ Close() error }, *genai.ChatSession, error) {
			return NewChatSession(ctx, record.ChatModel, gs.cfg)
		}, nil)
		conv := NewConversationFromRecord(repo, record)
		repo.onChange = func(_ []Message) {
			conv.Touch()
			gs.persistConversation(conv)
		}
		gs.cm.AddConversation(conv)
	}

	return nil
}

func (gs *GeminiService) persistConversation(conv *Conversation) {
	if conv == nil || conv.Repo == nil {
		return
	}

	messages, err := conv.Repo.GetMessages()
	if err != nil && !errors.Is(err, ErrNoMessagesInHistory) && !errors.Is(err, ErrSessionNotInitialized) {
		gs.logger.Warnf("Persist conversation failed: %v", err)
		return
	}

	record := ConversationRecord{
		ID:                conv.ID,
		Description:       conv.GetDescription(),
		DescriptionLocked: conv.IsDescriptionLocked(),
		CreatedAt:         conv.CreatedAt,
		UpdatedAt:         conv.UpdatedAtSnapshot(),
		ChatModel:         conv.ChatModel,
		Messages:          messages,
	}
	if err := gs.store.Save(record); err != nil {
		gs.logger.Warnf("Persist conversation failed: %v", err)
	}
}

func (gs *GeminiService) RenameConversation(id string, description string) error {
	description = strings.TrimSpace(description)
	if description == "" {
		return fmt.Errorf("conversation title cannot be empty")
	}

	var target *Conversation
	for _, conv := range gs.cm.All() {
		if conv.ID == id {
			target = conv
			break
		}
	}
	if target == nil {
		return fmt.Errorf("conversation with ID %s does not exist", id)
	}

	target.SetDescription(description)
	target.SetDescriptionLocked(true)
	gs.persistConversation(target)
	return nil
}

func (gs *GeminiService) DeleteConversation(id string) error {
	if _, err := gs.cm.RemoveConversation(id); err != nil {
		return err
	}
	if err := gs.store.Delete(id); err != nil {
		return err
	}
	return nil
}

const maxDescriptionMessages = 6

func buildDescriptionPrompt(messages []Message) string {
	if len(messages) > maxDescriptionMessages {
		messages = messages[:maxDescriptionMessages]
	}

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

	if summary, ok := utils.SummarizeKnownError(err); ok {
		return prefix + ": " + summary
	}

	return prefix + ": request failed."
}
