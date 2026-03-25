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

// initialized to the current UTC time.
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

// NewConversationFromRecord creates a Conversation populated from the given ConversationRecord and repository.
// It normalizes timestamps and fills missing values: if record.CreatedAt is zero it uses the current UTC time,
// and if record.UpdatedAt is zero it uses the resolved CreatedAt. If record.Description is empty the conversation's
// description is set to "New Conversation..." and descriptionLocked is set to false; otherwise descriptionLocked is set to true.
// ID, Repo, and ChatModel are taken from the provided record and repo.
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
		descriptionLocked: description != "New Conversation...",
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
	// Keep the channel open for the lifetime of the conversation object.
	// Switching away from a conversation should not turn future receives
	// into zero-value reads when that conversation becomes active again.
}

// GenerateRandomConversationID generates a new conversation identifier.
// The identifier is formatted as "CONVERSATION-<HEX>", where <HEX> is an uppercase
// MD5 hex string derived from three randomly generated integers.
func GenerateRandomConversationID() string {

	randomString := fmt.Sprintf("%x-%x-%x", rand.Int(), rand.Int(), rand.Int())
	hash := md5.Sum([]byte(randomString))
	id_string := hex.EncodeToString(hash[:])
	return strings.ToUpper(fmt.Sprintf("CONVERSATION-%s", id_string))
}
