package appconfig

import (
	"os"
	"path/filepath"
	"strings"
	"testing"
)

func TestExpertInstructionAndProfessionalPromptUpgrade(t *testing.T) {
	cfg := &Config{SystemPrompt: professionalSystemPrompt}
	upgradeDefaultPrompts(cfg)
	if !strings.Contains(cfg.SystemPrompt, expertDecisionInstruction) {
		t.Fatal("expert decision instruction missing after upgrade")
	}
	cfg.SystemPrompt = "Custom system prompt"
	upgradeDefaultPrompts(cfg)
	if cfg.SystemPrompt != "Custom system prompt" {
		t.Fatal("custom prompt overwritten")
	}
}

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

func TestLoadUpgradesLegacyPrompts(t *testing.T) {
	t.Setenv("HOME", t.TempDir())
	cfg, err := Load()
	if err != nil {
		t.Fatal(err)
	}
	for path, prompt := range map[string]string{
		cfg.SystemPromptFile:      legacySystemPrompt,
		cfg.DescriptionPromptFile: legacyDescriptionPrompt,
	} {
		if err := os.WriteFile(path, []byte(prompt), 0644); err != nil {
			t.Fatal(err)
		}
	}
	cfg, err = Load()
	if err != nil {
		t.Fatal(err)
	}
	if cfg.SystemPrompt != defaultSystemPrompt || cfg.DescriptionPrompt != defaultDescriptionPrompt {
		t.Fatal("legacy prompts were not upgraded")
	}
}
