package gemini

import (
	"crypto/md5"
	"encoding/hex"
	"fmt"
	"math/rand/v2"
	"strings"
	"sync"
)

type Conversation struct {
	ID                string
	description       string
	Repo              HistoryRepository
	descriptionLocked bool

	mu sync.RWMutex
}

func NewConversation(repo HistoryRepository) *Conversation {
	c := &Conversation{
		ID:                GenerateRandomConversationID(),
		Repo:              repo,
		description:       "New Conversation...",
		descriptionLocked: false,
	}
	return c
}

func (c *Conversation) SetDescription(description string) {
	c.mu.Lock()
	defer c.mu.Unlock()
	c.description = description
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

func GenerateRandomConversationID() string {

	randomString := fmt.Sprintf("%x-%x-%x", rand.Int(), rand.Int(), rand.Int())
	hash := md5.Sum([]byte(randomString))
	id_string := hex.EncodeToString(hash[:])
	return strings.ToUpper(fmt.Sprintf("CONVERSATION-%s", id_string))
}
