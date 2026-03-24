package gemini

import (
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"sort"
	"time"
)

type ConversationRecord struct {
	ID          string    `json:"id"`
	Description string    `json:"description"`
	CreatedAt   time.Time `json:"created_at"`
	UpdatedAt   time.Time `json:"updated_at"`
	ChatModel   string    `json:"chat_model"`
	Messages    []Message `json:"messages"`
}

type FileConversationStore struct {
	dir string
}

func NewFileConversationStore(dataDir string) *FileConversationStore {
	return &FileConversationStore{dir: filepath.Join(dataDir, "conversations")}
}

func (s *FileConversationStore) Ensure() error {
	return os.MkdirAll(s.dir, 0755)
}

func (s *FileConversationStore) Save(record ConversationRecord) error {
	if err := s.Ensure(); err != nil {
		return err
	}
	record.UpdatedAt = record.UpdatedAt.UTC()
	record.CreatedAt = record.CreatedAt.UTC()

	data, err := json.MarshalIndent(record, "", "  ")
	if err != nil {
		return fmt.Errorf("marshal conversation record: %w", err)
	}

	return os.WriteFile(s.pathFor(record.ID), append(data, '\n'), 0644)
}

func (s *FileConversationStore) LoadAll() ([]ConversationRecord, error) {
	if err := s.Ensure(); err != nil {
		return nil, err
	}

	entries, err := os.ReadDir(s.dir)
	if err != nil {
		return nil, fmt.Errorf("read conversations dir: %w", err)
	}

	var records []ConversationRecord
	for _, entry := range entries {
		if entry.IsDir() || filepath.Ext(entry.Name()) != ".json" {
			continue
		}

		record, err := s.loadFile(filepath.Join(s.dir, entry.Name()))
		if err != nil {
			return nil, err
		}
		records = append(records, record)
	}

	sort.Slice(records, func(i, j int) bool {
		return records[i].UpdatedAt.After(records[j].UpdatedAt)
	})

	return records, nil
}

func (s *FileConversationStore) pathFor(id string) string {
	return filepath.Join(s.dir, id+".json")
}

func (s *FileConversationStore) loadFile(path string) (ConversationRecord, error) {
	data, err := os.ReadFile(path)
	if err != nil {
		return ConversationRecord{}, fmt.Errorf("read conversation file %s: %w", path, err)
	}

	var record ConversationRecord
	if err := json.Unmarshal(data, &record); err != nil {
		return ConversationRecord{}, fmt.Errorf("parse conversation file %s: %w", path, err)
	}

	return record, nil
}
