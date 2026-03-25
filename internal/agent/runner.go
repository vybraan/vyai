package agent

import (
	"context"
	"fmt"
	"strings"
)

type Translator func(context.Context, string, string) (string, error)

type RunRequest struct {
	Input string
	Model string
}

type Runner interface {
	Run(context.Context, RunRequest) (string, error)
}

type LocalRunner struct {
	workspace string
	translate Translator
}

func NewLocalRunner(workspace string, translate Translator) *LocalRunner {
	return &LocalRunner{
		workspace: workspace,
		translate: translate,
	}
}

func (r *LocalRunner) Run(ctx context.Context, req RunRequest) (string, error) {
	userInput := strings.TrimSpace(req.Input)
	if userInput == "" {
		return "", fmt.Errorf("usage: /agent <prompt>")
	}

	agentInput := userInput
	if !LooksLikePromptWeaverInput(userInput) {
		if r.translate == nil {
			return "", fmt.Errorf("agent translation is not configured")
		}

		translated, err := r.translate(ctx, req.Model, BuildTranslationPrompt(userInput))
		if err != nil {
			return "", fmt.Errorf("translate agent request: %w", err)
		}

		agentInput = strings.TrimSpace(translated)
		if !LooksLikePromptWeaverInput(agentInput) {
			return "", fmt.Errorf("translation did not produce a valid tool program")
		}
	}

	var output []string
	engine := NewAgent(func(line string) {
		line = strings.TrimSpace(line)
		if line != "" {
			output = append(output, line)
		}
	}, r.workspace)

	if err := engine.Process(strings.NewReader(agentInput)); err != nil {
		return "", err
	}
	if len(output) == 0 {
		return "", fmt.Errorf("agent completed with no visible output")
	}

	return strings.Join(output, "\n\n"), nil
}

var promptWeaverTags = map[string]struct{}{
	"think":       {},
	"run-bash":    {},
	"create-file": {},
	"read-file":   {},
	"list-dir":    {},
	"grep-file":   {},
	"glob-file":   {},
	"edit-file":   {},
	"summary":     {},
}

func LooksLikePromptWeaverInput(input string) bool {
	for _, part := range strings.Split(input, "<") {
		part = strings.TrimSpace(part)
		if part == "" {
			continue
		}
		part = strings.TrimPrefix(part, "/")

		end := len(part)
		for i, r := range part {
			if r == '>' || r == ' ' || r == '\n' || r == '\r' || r == '\t' {
				end = i
				break
			}
		}
		if _, ok := promptWeaverTags[part[:end]]; ok {
			return true
		}
	}
	return false
}

func BuildTranslationPrompt(userInput string) string {
	return strings.TrimSpace(fmt.Sprintf(`
You convert a user's natural-language request into PromptWeaver sections for a local coding agent.

Output rules:
- Reply with PromptWeaver tags only.
- Do not use markdown fences.
- Do not explain what you are doing outside tags.
- Use only these tags:
  - <think>hidden reasoning or short plan</think>
  - <run-bash>safe shell command</run-bash>
  - <create-file path="...">content</create-file>
  - <read-file path="..."></read-file>
  - <list-dir path="..."></list-dir>
  - <grep-file path="..." pattern="..." include="..."></grep-file>
  - <glob-file path="..." pattern="..."></glob-file>
  - <edit-file path="..." old="..." new="..."></edit-file>
  - <summary>final visible response</summary>
- Prefer read-only actions unless the user clearly asks to modify files.
- Always end with exactly one <summary>...</summary>.
- If the task is unclear or cannot be completed safely, emit only a <summary> explaining what is missing.

User request:
%s
`, userInput))
}
