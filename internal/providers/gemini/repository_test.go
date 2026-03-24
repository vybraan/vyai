package gemini

import (
	"testing"

	"github.com/google/generative-ai-go/genai"
)

func TestMemoryHistoryRepositoryGetMessagesReturnsTypedMessages(t *testing.T) {
	t.Parallel()

	repo := NewMemoryHistoryRepository(&genai.ChatSession{
		History: []*genai.Content{
			{
				Role:  "user",
				Parts: []genai.Part{genai.Text("hello")},
			},
			{
				Role:  "model",
				Parts: []genai.Part{genai.Text("world")},
			},
		},
	})

	messages, err := repo.GetMessages()
	if err != nil {
		t.Fatalf("GetMessages returned error: %v", err)
	}

	if len(messages) != 2 {
		t.Fatalf("expected 2 messages, got %d", len(messages))
	}

	if messages[0] != (Message{Role: "user", Text: "hello"}) {
		t.Fatalf("unexpected first message: %#v", messages[0])
	}

	if messages[1] != (Message{Role: "model", Text: "world"}) {
		t.Fatalf("unexpected second message: %#v", messages[1])
	}
}

func TestBuildDescriptionPromptIncludesRoles(t *testing.T) {
	t.Parallel()

	prompt := buildDescriptionPrompt([]Message{
		{Role: "user", Text: "first question"},
		{Role: "model", Text: "first answer"},
	})

	expected := "[user] first question\n[model] first answer"
	if prompt != expected {
		t.Fatalf("unexpected prompt: %q", prompt)
	}
}
