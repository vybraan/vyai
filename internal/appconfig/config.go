package appconfig

import (
	"encoding/json"
	"errors"
	"fmt"
	"os"
	"path/filepath"
	"strings"
)

const (
	DefaultAppName              = "vyai"
	DefaultChatModel            = "gemini-3-flash-preview"
	DefaultDescriptionModel     = "gemini-3-flash-preview"
	DefaultSystemPromptFileName = "system_prompt.md"
	DefaultTitlePromptFileName  = "description_prompt.md"
	DefaultConfigFileName       = "config.json"
	defaultSystemPrompt         = `
You are a Linux System Admin Assistant. Your role is to assist with Linux and infrastructure management by providing clear, concise, and direct answers. Focus on actionable guidance for:

- Linux commands and scripting
- System administration tasks
- Networking and security best practices
- Programming concepts and languages
- Troubleshooting and problem-solving
- Provide descriptions
- Provide summaries

Always prioritize clarity and brevity. Use markdown formatting for all responses, including:

- Code examples (use code blocks)
- Lists (use bullet points)
- Step-by-step guides (use headings)
- Summaries, comparisons, definitions, and explanations (use headings)
- Solutions, recommendations, and suggestions (use headings)

Your name: vyai (vybraan artificial intelligence)
Creator: vybraan
`
	defaultDescriptionPrompt = `
Please give a description to this conversation. Reply only with the description. Do not include any other text. For example:

- 'REST Compliance vs HTTP'
- 'Request for Clarification'
- 'Banner design request'
- 'Who is John Clan'
- 'Processo vs Sistema'
- 'Unexpected Story Twist'
- 'Brutal Dev Roast'

Always use the first messages to describe the conversation. Do not use the last message to describe the conversation.

Words - Max: 15, Min: 3, Recommended Max: 10`
)

type fileConfig struct {
	ChatModel             string `json:"chat_model"`
	DescriptionModel      string `json:"description_model"`
	SystemPromptFile      string `json:"system_prompt_file"`
	DescriptionPromptFile string `json:"description_prompt_file"`
	DataDir               string `json:"data_dir"`
}

type Config struct {
	AppName               string
	ConfigDir             string
	ConfigFile            string
	DataDir               string
	ChatModel             string
	DescriptionModel      string
	SystemPrompt          string
	DescriptionPrompt     string
	SystemPromptFile      string
	DescriptionPromptFile string
	SystemPromptSource    string
	DescriptionSource     string
}

func Load() (*Config, error) {
	home, err := os.UserHomeDir()
	if err != nil {
		return nil, fmt.Errorf("resolve home directory: %w", err)
	}

	cfgDir := filepath.Join(home, ".config", "vybr", DefaultAppName)
	cfg := &Config{
		AppName:               DefaultAppName,
		ConfigDir:             cfgDir,
		ConfigFile:            filepath.Join(cfgDir, DefaultConfigFileName),
		DataDir:               filepath.Join(home, ".vybr", DefaultAppName),
		ChatModel:             DefaultChatModel,
		DescriptionModel:      DefaultDescriptionModel,
		SystemPrompt:          strings.TrimSpace(defaultSystemPrompt),
		DescriptionPrompt:     strings.TrimSpace(defaultDescriptionPrompt),
		SystemPromptFile:      filepath.Join(cfgDir, DefaultSystemPromptFileName),
		DescriptionPromptFile: filepath.Join(cfgDir, DefaultTitlePromptFileName),
		SystemPromptSource:    "built-in default",
		DescriptionSource:     "built-in default",
	}

	if err := bootstrapDefaults(cfg); err != nil {
		return nil, err
	}
	if err := applyFileConfig(cfg); err != nil {
		return nil, err
	}
	if err := os.MkdirAll(cfg.DataDir, 0700); err != nil {
		return nil, fmt.Errorf("create data dir: %w", err)
	}
	if err := os.Chmod(cfg.DataDir, 0700); err != nil {
		return nil, fmt.Errorf("chmod data dir: %w", err)
	}
	if err := loadPromptFile(&cfg.SystemPrompt, &cfg.SystemPromptSource, cfg.SystemPromptFile); err != nil {
		return nil, err
	}
	if err := loadPromptFile(&cfg.DescriptionPrompt, &cfg.DescriptionSource, cfg.DescriptionPromptFile); err != nil {
		return nil, err
	}

	return cfg, nil
}

func bootstrapDefaults(cfg *Config) error {
	if err := os.MkdirAll(cfg.ConfigDir, 0755); err != nil {
		return fmt.Errorf("create config dir: %w", err)
	}
	if err := writeFileIfMissing(cfg.ConfigFile, defaultConfigJSON(cfg)); err != nil {
		return fmt.Errorf("bootstrap config file: %w", err)
	}
	if err := writeFileIfMissing(cfg.SystemPromptFile, cfg.SystemPrompt+"\n"); err != nil {
		return fmt.Errorf("bootstrap system prompt: %w", err)
	}
	if err := writeFileIfMissing(cfg.DescriptionPromptFile, cfg.DescriptionPrompt+"\n"); err != nil {
		return fmt.Errorf("bootstrap description prompt: %w", err)
	}

	return nil
}

func applyFileConfig(cfg *Config) error {
	data, err := os.ReadFile(cfg.ConfigFile)
	if err != nil {
		if errors.Is(err, os.ErrNotExist) {
			return nil
		}
		return fmt.Errorf("read config file: %w", err)
	}

	var fc fileConfig
	if err := json.Unmarshal(data, &fc); err != nil {
		return fmt.Errorf("parse config file: %w", err)
	}

	if fc.ChatModel != "" {
		cfg.ChatModel = fc.ChatModel
	}
	if fc.DescriptionModel != "" {
		cfg.DescriptionModel = fc.DescriptionModel
	}
	if fc.DataDir != "" {
		cfg.DataDir = expandPath(fc.DataDir, cfg.ConfigDir)
	}
	if fc.SystemPromptFile != "" {
		cfg.SystemPromptFile = expandPath(fc.SystemPromptFile, cfg.ConfigDir)
	}
	if fc.DescriptionPromptFile != "" {
		cfg.DescriptionPromptFile = expandPath(fc.DescriptionPromptFile, cfg.ConfigDir)
	}

	return nil
}

func loadPromptFile(target *string, source *string, path string) error {
	data, err := os.ReadFile(path)
	if err != nil {
		if errors.Is(err, os.ErrNotExist) {
			return nil
		}
		return fmt.Errorf("read prompt file %s: %w", path, err)
	}

	content := strings.TrimSpace(string(data))
	if content == "" {
		return nil
	}

	*target = content
	*source = path
	return nil
}

func writeFileIfMissing(path string, content string) error {
	if _, err := os.Stat(path); err == nil {
		return nil
	} else if !errors.Is(err, os.ErrNotExist) {
		return err
	}

	return os.WriteFile(path, []byte(content), 0644)
}

func defaultConfigJSON(cfg *Config) string {
	return fmt.Sprintf(`{
  "chat_model": %q,
  "description_model": %q,
  "system_prompt_file": %q,
  "description_prompt_file": %q,
  "data_dir": %q
}
`, cfg.ChatModel, cfg.DescriptionModel, DefaultSystemPromptFileName, DefaultTitlePromptFileName, cfg.DataDir)
}

func expandPath(path string, baseDir string) string {
	if path == "" {
		return path
	}
	if strings.HasPrefix(path, "~/") {
		home, err := os.UserHomeDir()
		if err == nil {
			return filepath.Join(home, strings.TrimPrefix(path, "~/"))
		}
		return filepath.Join(baseDir, strings.TrimPrefix(path, "~/"))
	}
	if filepath.IsAbs(path) {
		return path
	}
	return filepath.Join(baseDir, path)
}
