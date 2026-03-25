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

Your name: vyai (vybraan artificial inteligence)
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

// Load constructs and returns the resolved application configuration by combining built-in defaults,
// optional overrides from a config.json, and optional prompt contents from configured prompt files.
// Load resolves the user's home directory, establishes default config/data/prompt paths under the
// user's home, ensures required directories and default files exist, applies non-empty overrides from
// the config file (if present), ensures the data directory exists, and replaces in-memory prompts when
// prompt files exist and contain non-empty trimmed content (recording the source for each prompt).
// It returns an error if the home directory cannot be resolved or if any filesystem or JSON read/write
// operation required during bootstrapping, override application, or prompt loading fails.
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
	if err := os.MkdirAll(cfg.DataDir, 0755); err != nil {
		return nil, fmt.Errorf("create data dir: %w", err)
	}
	if err := loadPromptFile(&cfg.SystemPrompt, &cfg.SystemPromptSource, cfg.SystemPromptFile); err != nil {
		return nil, err
	}
	if err := loadPromptFile(&cfg.DescriptionPrompt, &cfg.DescriptionSource, cfg.DescriptionPromptFile); err != nil {
		return nil, err
	}

	return cfg, nil
}

// bootstrapDefaults ensures the configuration directory exists and that the default
// config file, system prompt file, and description prompt file are present, creating
// them with default contents when missing. It returns an error if directory creation
// or any file write fails (errors include contextual wrapping).
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

// applyFileConfig loads overrides from cfg.ConfigFile and applies any non-empty fields
// to the provided cfg.
//
// If the config file does not exist the function is a no-op. On success it updates
// ChatModel and DescriptionModel directly; for DataDir, SystemPromptFile and
// DescriptionPromptFile it expands paths (supporting `~/` and relative paths using
// cfg.ConfigDir as the base) before assigning. Read and JSON parse errors are
// returned with context.
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

// loadPromptFile reads the file at path and, if it exists and contains non-empty trimmed content, sets target to that content and source to the path.
// If the file does not exist or the trimmed content is empty, target and source are left unchanged.
// Returns a wrapped error for file read failures other than non-existence.
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

// writeFileIfMissing creates the file at path containing content if the file does not already exist.
// If the file exists, it returns nil without modifying it. It returns any error encountered while
// checking existence or writing the file.
func writeFileIfMissing(path string, content string) error {
	if _, err := os.Stat(path); err == nil {
		return nil
	} else if !errors.Is(err, os.ErrNotExist) {
		return err
	}

	return os.WriteFile(path, []byte(content), 0644)
}

// defaultConfigJSON generates a JSON-formatted default configuration using cfg's
// chat model, description model, and data directory; prompt file names use the
// package default constants. The produced string is suitable for writing to the
// default config file.
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

// expandPath expands a given path: it replaces a leading "~/" with the user's home directory when available, leaves absolute paths unchanged, and for other non-empty relative paths joins them with baseDir.
// If the path is empty it is returned unchanged; if home directory lookup fails the "~/" prefix is not expanded.
func expandPath(path string, baseDir string) string {
	if path == "" {
		return path
	}
	if strings.HasPrefix(path, "~/") {
		home, err := os.UserHomeDir()
		if err == nil {
			return filepath.Join(home, strings.TrimPrefix(path, "~/"))
		}
	}
	if filepath.IsAbs(path) {
		return path
	}
	return filepath.Join(baseDir, path)
}
