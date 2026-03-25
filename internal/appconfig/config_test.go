package appconfig

import (
	"os"
	"path/filepath"
	"testing"
)

func TestLoadUsesVybrPathsByDefault(t *testing.T) {
	home := t.TempDir()
	t.Setenv("HOME", home)

	cfg, err := Load()
	if err != nil {
		t.Fatalf("Load returned error: %v", err)
	}

	if cfg.ConfigDir != filepath.Join(home, ".config", "vybr", "vyai") {
		t.Fatalf("unexpected config dir: %s", cfg.ConfigDir)
	}
	if cfg.DataDir != filepath.Join(home, ".vybr", "vyai") {
		t.Fatalf("unexpected data dir: %s", cfg.DataDir)
	}
	for _, path := range []string{cfg.ConfigFile, cfg.SystemPromptFile, cfg.DescriptionPromptFile, cfg.DataDir} {
		if _, err := os.Stat(path); err != nil {
			t.Fatalf("expected bootstrap path %s to exist: %v", path, err)
		}
	}
}

func TestLoadReadsPromptFilesAndModels(t *testing.T) {
	home := t.TempDir()
	t.Setenv("HOME", home)

	configDir := filepath.Join(home, ".config", "vybr", "vyai")
	if err := os.MkdirAll(configDir, 0755); err != nil {
		t.Fatalf("mkdir config dir: %v", err)
	}

	if err := os.WriteFile(filepath.Join(configDir, "config.json"), []byte(`{
  "chat_model": "gemini-custom-chat",
  "description_model": "gemini-custom-title",
  "system_prompt_file": "prompt.txt",
  "description_prompt_file": "title.txt",
  "data_dir": "~/.vybr/vygrant"
}`), 0644); err != nil {
		t.Fatalf("write config file: %v", err)
	}
	if err := os.WriteFile(filepath.Join(configDir, "prompt.txt"), []byte("system prompt override"), 0644); err != nil {
		t.Fatalf("write system prompt: %v", err)
	}
	if err := os.WriteFile(filepath.Join(configDir, "title.txt"), []byte("description prompt override"), 0644); err != nil {
		t.Fatalf("write description prompt: %v", err)
	}

	cfg, err := Load()
	if err != nil {
		t.Fatalf("Load returned error: %v", err)
	}

	if cfg.ChatModel != "gemini-custom-chat" {
		t.Fatalf("unexpected chat model: %s", cfg.ChatModel)
	}
	if cfg.DescriptionModel != "gemini-custom-title" {
		t.Fatalf("unexpected description model: %s", cfg.DescriptionModel)
	}
	if cfg.SystemPrompt != "system prompt override" {
		t.Fatalf("unexpected system prompt: %q", cfg.SystemPrompt)
	}
	if cfg.DescriptionPrompt != "description prompt override" {
		t.Fatalf("unexpected description prompt: %q", cfg.DescriptionPrompt)
	}
	if cfg.DataDir != filepath.Join(home, ".vybr", "vygrant") {
		t.Fatalf("unexpected data dir: %s", cfg.DataDir)
	}
}
