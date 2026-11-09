package gemini

import "testing"

func TestMessageHistoryRoundTrip(t *testing.T) {
	want := []Message{
		{Role: "user", Text: "hello"},
		{Role: "model", Text: "world"},
	}
	got := messagesFromHistory(historyFromMessages(want))
	if len(got) != len(want) {
		t.Fatalf("expected %d messages, got %d", len(want), len(got))
	}
	for i := range want {
		if got[i] != want[i] {
			t.Fatalf("message %d: got %#v, want %#v", i, got[i], want[i])
		}
	}
}

func TestMemoryHistoryRepositoryGetMessagesReturnsTypedMessages(t *testing.T) {
	t.Parallel()

	repo := NewPersistentHistoryRepository([]Message{
		{Role: "user", Text: "hello"},
		{Role: "model", Text: "world"},
	}, nil, nil)

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

func TestMemoryHistoryRepositoryResetSessionPreservesMessages(t *testing.T) {
	t.Parallel()

	repo := NewPersistentHistoryRepository([]Message{
		{Role: "user", Text: "hello"},
		{Role: "model", Text: "world"},
	}, nil, nil)
	repo.ResetSession()

	if repo.chatSession != nil {
		t.Fatal("expected chat session to be cleared")
	}

	messages, err := repo.GetMessages()
	if err != nil {
		t.Fatalf("GetMessages returned error: %v", err)
	}
	if len(messages) != 2 {
		t.Fatalf("expected 2 cached messages, got %d", len(messages))
	}
}
