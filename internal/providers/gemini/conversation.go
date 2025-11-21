package gemini

import (
	"crypto/md5"
	"encoding/hex"
	"fmt"
	"math/rand/v2"
	"strings"
)

type Conversation struct {
	ID                 string
	Description        string
	Repo               HistoryRepository
	DescriptionLocked  bool
	DescriptionChannel chan string
}

func NewConversation(repo HistoryRepository) *Conversation {
	return &Conversation{
		ID:                 GenerateRandomConversationID(),
		Repo:               repo,
		Description:        "New Conversation...",
		DescriptionLocked:  false,
		DescriptionChannel: make(chan string, 1),
	}
}

func (c *Conversation) SetDescription(description string) {
	c.Description = description
}

func GenerateRandomConversationID() string {

	randomString := fmt.Sprintf("%x-%x-%x", rand.Int(), rand.Int(), rand.Int())
	hash := md5.Sum([]byte(randomString))
	id_string := hex.EncodeToString(hash[:])
	return strings.ToUpper(fmt.Sprintf("CONVERSATION-%s", id_string))
}
