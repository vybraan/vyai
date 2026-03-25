package gemini

import (
	"os"
	"path/filepath"
	"testing"
	"time"
)

func TestFileConversationStoreSaveAndLoadAll(t *testing.T) {
	t.Parallel()

	store := NewFileConversationStore(t.TempDir())
	record := ConversationRecord{
		ID:          "CONVERSATION-1",
		Description: "Example",
		CreatedAt:   time.Now().UTC().Add(-time.Minute),
		UpdatedAt:   time.Now().UTC(),
		ChatModel:   "gemini-test",
		Messages: []Message{
			{Role: "user", Text: "hello"},
			{Role: "model", Text: "world"},
		},
	}

	if err := store.Save(record); err != nil {
		t.Fatalf("Save returned error: %v", err)
	}

	records, err := store.LoadAll()
	if err != nil {
		t.Fatalf("LoadAll returned error: %v", err)
	}
	if len(records) != 1 {
		t.Fatalf("expected 1 record, got %d", len(records))
	}
	if records[0].ID != record.ID {
		t.Fatalf("unexpected record id: %s", records[0].ID)
	}
	if len(records[0].Messages) != 2 {
		t.Fatalf("expected 2 messages, got %d", len(records[0].Messages))
	}
}

func TestFileConversationStoreUsesConversationsDir(t *testing.T) {
	t.Parallel()

	root := t.TempDir()
	store := NewFileConversationStore(root)
	if err := store.Ensure(); err != nil {
		t.Fatalf("Ensure returned error: %v", err)
	}

	expected := filepath.Join(root, "conversations")
	if _, err := os.Stat(expected); err != nil {
		t.Fatalf("expected conversations dir %s to exist: %v", expected, err)
	}
}

func TestFileConversationStoreDeleteRemovesRecord(t *testing.T) {
	t.Parallel()

	store := NewFileConversationStore(t.TempDir())
	record := ConversationRecord{
		ID:          "CONVERSATION-DEADBEEF",
		Description: "Delete me",
		CreatedAt:   time.Now().UTC().Add(-time.Minute),
		UpdatedAt:   time.Now().UTC(),
	}

	if err := store.Save(record); err != nil {
		t.Fatalf("Save returned error: %v", err)
	}
	if err := store.Delete(record.ID); err != nil {
		t.Fatalf("Delete returned error: %v", err)
	}

	records, err := store.LoadAll()
	if err != nil {
		t.Fatalf("LoadAll returned error: %v", err)
	}
	if len(records) != 0 {
		t.Fatalf("expected no records after delete, got %d", len(records))
	}
}
