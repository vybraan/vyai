package agent

import (
	"context"
	"fmt"
	"regexp"
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

// NewLocalRunner creates a LocalRunner configured with the given workspace and optional Translator.
// The returned runner uses workspace for the underlying agent engine and, if non-nil, uses translate to convert natural-language prompts into PromptWeaver-formatted input.
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

var promptWeaverTagPattern = regexp.MustCompile(`(?s)<\s*/?\s*([a-zA-Z][a-zA-Z0-9_-]*)`)

// LooksLikePromptWeaverInput reports whether the input contains PromptWeaver-style tags (for example `<tag>` or `</tag>`).
// It returns true if the string contains a tag-like sequence starting with `<`, optionally `/`, followed by a tag name.
func LooksLikePromptWeaverInput(input string) bool {
	return promptWeaverTagPattern.MatchString(input)
}

// BuildTranslationPrompt builds a strict instruction prompt that directs a translator to convert a natural-language user request into PromptWeaver-formatted tags.
// The prompt requires the translator to emit only PromptWeaver tags (no markdown fences or explanations outside tags), limits output to an explicit allowlist of tags
// (<think>, <run-bash>, <create-file path="...">, <read-file path="...">, <list-dir path="...">, <grep-file path="..." pattern="..." include="...">, <glob-file path="..." pattern="...">, <edit-file path="..." old="..." new="...">, and <summary>), prefers read-only actions unless the user explicitly requests modifications, and mandates exactly one final <summary>...</summary> (or only a <summary> describing missing/unsafe/unclear requirements).
// The user's request is placed after "User request:" and the returned string is trimmed of surrounding whitespace.
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
