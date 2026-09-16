package gemini

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"os"
	"strings"
	"sync"
	"time"

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
	descriptionUpdates chan DescriptionUpdate
	notices            chan Notice
	titleMu            sync.Mutex
	saveMu             sync.Mutex
	foreground         int
	idleSince          time.Time
	titleCancel        context.CancelFunc
	titleDone          chan struct{}
	generateTitle      func(context.Context, string, string) (string, error)
}

func NewGeminiService(cm *ConversationManager, cfg *appconfig.Config) *GeminiService {
	return &GeminiService{
		cm:                 cm,
		cfg:                cfg,
		store:              NewFileConversationStore(cfg.DataDir),
		descriptionUpdates: make(chan DescriptionUpdate, 8),
		notices:            make(chan Notice, 8),
		generateTitle:      utils.GenerateEphemeralMessage,
		idleSince:          time.Now(),
	}
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
	release, err := gs.BeginForeground(c)
	if err != nil {
		return "", err
	}
	defer release()

	conversation, err := gs.cm.GetActiveConversation()
	if err != nil {
		conversation, err = gs.NewConversation(c)
		if err != nil {
			return "", err
		}
	}

	gs.queueTitle(conversation, []Message{{Role: "user", Text: message}})
	result, err := conversation.Repo.SendMessage(c, genai.Text(message))

	if err != nil {
		return "", err
	}
	conversation.Touch()

	return result, nil
}

func (gs *GeminiService) SendMessageStream(c context.Context, message string, onToken func(string)) (string, error) {
	release, err := gs.BeginForeground(c)
	if err != nil {
		return "", err
	}
	defer release()
	conversation, err := gs.cm.GetActiveConversation()
	if err != nil {
		conversation, err = gs.NewConversation(c)
		if err != nil {
			return "", err
		}
	}

	gs.queueTitle(conversation, []Message{{Role: "user", Text: message}})
	result, err := conversation.Repo.SendMessageStream(c, genai.Text(message), onToken)
	if err != nil {
		return "", err
	}
	conversation.Touch()

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
	gs.titleMu.Lock()
	defer gs.titleMu.Unlock()
	oldCfg := gs.cfg

	cfg, err := appconfig.Load()
	if err != nil {
		return err
	}

	gs.cfg = cfg
	gs.saveMu.Lock()
	gs.store = NewFileConversationStore(cfg.DataDir)
	gs.saveMu.Unlock()
	for _, conv := range gs.cm.All() {
		conv.mu.Lock()
		if conv.ChatModel == oldCfg.ChatModel {
			conv.ChatModel = cfg.ChatModel
		}
		conv.mu.Unlock()
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
		var conv *Conversation
		repo := NewPersistentHistoryRepository(record.Messages, func(ctx context.Context) (interface{ Close() error }, *genai.ChatSession, error) {
			modelID := record.ChatModel
			if conv != nil && conv.ChatModel != "" {
				modelID = conv.ChatModel
			}
			return NewChatSession(ctx, modelID, gs.cfg)
		}, nil)
		conv = NewConversationFromRecord(repo, record)
		repo.onChange = func(_ []Message) {
			conv.Touch()
			gs.persistConversation(conv)
		}
		gs.cm.AddConversation(conv)
		gs.queueTitle(conv, record.Messages)
	}

	return nil
}

func (gs *GeminiService) persistConversation(conv *Conversation) {
	gs.saveMu.Lock()
	defer gs.saveMu.Unlock()
	gs.saveConversation(conv)
}

func (gs *GeminiService) saveConversation(conv *Conversation) {
	if conv == nil || conv.Repo == nil {
		return
	}
	gs.cm.mu.RLock()
	exists := gs.cm.conversations[conv.ID] == conv
	gs.cm.mu.RUnlock()
	if !exists {
		return
	}

	messages, err := conv.Repo.GetMessages()
	if err != nil && !errors.Is(err, ErrNoMessagesInHistory) && !errors.Is(err, ErrSessionNotInitialized) {
		return
	}

	conv.mu.RLock()
	record := ConversationRecord{
		ID:                conv.ID,
		Description:       conv.description,
		DescriptionLocked: conv.descriptionLocked,
		CreatedAt:         conv.CreatedAt,
		UpdatedAt:         conv.UpdatedAt,
		ChatModel:         conv.ChatModel,
		Messages:          messages,
		Title:             conv.title,
	}
	conv.mu.RUnlock()
	if err := gs.store.Save(record); err != nil {
		gs.publishNotice("Conversation could not be saved: " + err.Error())
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

	target.mu.Lock()
	target.description = description
	target.descriptionLocked = true
	target.title = TitleState{Manual: true}
	target.UpdatedAt = time.Now().UTC()
	target.mu.Unlock()
	gs.persistConversation(target)
	return nil
}

func (gs *GeminiService) SetChatModel(model string) error {
	old := gs.cfg.ChatModel
	gs.cfg.ChatModel = model
	if err := gs.persistConfig(); err != nil {
		gs.cfg.ChatModel = old
		return err
	}
	for _, conv := range gs.cm.All() {
		conv.mu.Lock()
		changed := conv.ChatModel == old || conv.ChatModel == ""
		if changed {
			conv.ChatModel = model
		}
		conv.mu.Unlock()
		if changed {
			conv.Repo.ResetSession()
			gs.persistConversation(conv)
		}
	}
	return nil
}

func (gs *GeminiService) SetDescriptionModel(model string) error {
	gs.titleMu.Lock()
	defer gs.titleMu.Unlock()
	gs.cfg.DescriptionModel = model
	return gs.persistConfig()
}

func (gs *GeminiService) persistConfig() error {
	cfg := gs.cfg
	fc := struct {
		ChatModel             string `json:"chat_model"`
		DescriptionModel      string `json:"description_model"`
		SystemPromptFile      string `json:"system_prompt_file"`
		DescriptionPromptFile string `json:"description_prompt_file"`
		DataDir               string `json:"data_dir"`
	}{
		ChatModel:             cfg.ChatModel,
		DescriptionModel:      cfg.DescriptionModel,
		SystemPromptFile:      cfg.SystemPromptFile,
		DescriptionPromptFile: cfg.DescriptionPromptFile,
		DataDir:               cfg.DataDir,
	}
	data, err := json.MarshalIndent(fc, "", "  ")
	if err != nil {
		return err
	}
	return os.WriteFile(cfg.ConfigFile, data, 0644)
}

func (gs *GeminiService) DeleteConversation(id string) error {
	gs.saveMu.Lock()
	defer gs.saveMu.Unlock()
	conversation, err := gs.cm.RemoveConversation(id)
	if err != nil {
		return err
	}
	conversation.Close()
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
