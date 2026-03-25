package gemini

import (
	"crypto/md5"
	"encoding/hex"
	"fmt"
	"math/rand/v2"
	"strings"
	"sync"
	"time"
)

type Conversation struct {
	ID                string
	description       string
	Repo              HistoryRepository
	descriptionLocked bool
	ChatModel         string
	CreatedAt         time.Time
	UpdatedAt         time.Time

	mu sync.RWMutex
}

func (c *Conversation) UpdatedAtSnapshot() time.Time {
	c.mu.RLock()
	defer c.mu.RUnlock()
	return c.UpdatedAt
}

func NewConversation(repo HistoryRepository, chatModel string) *Conversation {
	now := time.Now().UTC()
	c := &Conversation{
		ID:                GenerateRandomConversationID(),
		Repo:              repo,
		description:       "New Conversation...",
		descriptionLocked: false,
		ChatModel:         chatModel,
		CreatedAt:         now,
		UpdatedAt:         now,
	}
	return c
}

func NewConversationFromRecord(repo HistoryRepository, record ConversationRecord) *Conversation {
	createdAt := record.CreatedAt.UTC()
	if createdAt.IsZero() {
		createdAt = time.Now().UTC()
	}
	updatedAt := record.UpdatedAt.UTC()
	if updatedAt.IsZero() {
		updatedAt = createdAt
	}

	description := record.Description
	if description == "" {
		description = "New Conversation..."
	}

	return &Conversation{
		ID:                record.ID,
		Repo:              repo,
		description:       description,
		descriptionLocked: record.DescriptionLocked,
		ChatModel:         record.ChatModel,
		CreatedAt:         createdAt,
		UpdatedAt:         updatedAt,
	}
}

func (c *Conversation) SetDescription(description string) {
	c.mu.Lock()
	defer c.mu.Unlock()
	c.description = description
	c.UpdatedAt = time.Now().UTC()
}

func (c *Conversation) GetDescription() string {
	c.mu.RLock()
	defer c.mu.RUnlock()
	return c.description
}

func (c *Conversation) SetDescriptionLocked(locked bool) {
	c.mu.Lock()
	defer c.mu.Unlock()
	c.descriptionLocked = locked
}

func (c *Conversation) Touch() {
	c.mu.Lock()
	defer c.mu.Unlock()
	c.UpdatedAt = time.Now().UTC()
}

func (c *Conversation) IsDescriptionLocked() bool {
	c.mu.RLock()
	defer c.mu.RUnlock()
	return c.descriptionLocked
}

func (c *Conversation) Close() {
	if c.Repo != nil {
		c.Repo.ResetSession()
	}
}

func GenerateRandomConversationID() string {

	randomString := fmt.Sprintf("%x-%x-%x", rand.Int(), rand.Int(), rand.Int())
	hash := md5.Sum([]byte(randomString))
	id_string := hex.EncodeToString(hash[:])
	return strings.ToUpper(fmt.Sprintf("CONVERSATION-%s", id_string))
}
