package gemini

import (
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"regexp"
	"sort"
	"time"
)

type ConversationRecord struct {
	ID                string    `json:"id"`
	Description       string    `json:"description"`
	DescriptionLocked bool      `json:"description_locked"`
	CreatedAt         time.Time `json:"created_at"`
	UpdatedAt         time.Time `json:"updated_at"`
	ChatModel         string    `json:"chat_model"`
	Messages          []Message `json:"messages"`
}

type FileConversationStore struct {
	dir string
}

var conversationIDPattern = regexp.MustCompile(`^CONVERSATION-[A-F0-9]+$`)

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

	path, err := s.pathFor(record.ID)
	if err != nil {
		return err
	}

	tempFile, err := os.CreateTemp(s.dir, ".conversation-*.json")
	if err != nil {
		return fmt.Errorf("create temp conversation file: %w", err)
	}
	tempPath := tempFile.Name()
	defer os.Remove(tempPath)

	if _, err := tempFile.Write(append(data, '\n')); err != nil {
		tempFile.Close()
		return fmt.Errorf("write temp conversation file: %w", err)
	}
	if err := tempFile.Chmod(0644); err != nil {
		tempFile.Close()
		return fmt.Errorf("chmod temp conversation file: %w", err)
	}
	if err := tempFile.Sync(); err != nil {
		tempFile.Close()
		return fmt.Errorf("sync temp conversation file: %w", err)
	}
	if err := tempFile.Close(); err != nil {
		return fmt.Errorf("close temp conversation file: %w", err)
	}

	if err := os.Rename(tempPath, path); err != nil {
		return fmt.Errorf("replace conversation file: %w", err)
	}

	return nil
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

func (s *FileConversationStore) pathFor(id string) (string, error) {
	if err := validateConversationID(id); err != nil {
		return "", err
	}
	return filepath.Join(s.dir, id+".json"), nil
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

func (s *FileConversationStore) Delete(id string) error {
	if err := s.Ensure(); err != nil {
		return err
	}

	path, err := s.pathFor(id)
	if err != nil {
		return err
	}
	if err := os.Remove(path); err != nil && !os.IsNotExist(err) {
		return fmt.Errorf("delete conversation file %s: %w", path, err)
	}

	return nil
}

func validateConversationID(id string) error {
	if !conversationIDPattern.MatchString(id) {
		return fmt.Errorf("invalid conversation ID: %s", id)
	}
	return nil
}
