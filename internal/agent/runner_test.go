package agent

import (
	"context"
	"strings"
	"testing"
)

func TestLooksLikePromptWeaverInput(t *testing.T) {
	t.Parallel()

	if !LooksLikePromptWeaverInput("<summary>done</summary>") {
		t.Fatal("expected PromptWeaver input to be detected")
	}
	if LooksLikePromptWeaverInput("list the files in this repo") {
		t.Fatal("expected plain English input not to be detected")
	}
}

func TestBuildTranslationPromptIncludesUserRequestAndTags(t *testing.T) {
	t.Parallel()

	prompt := BuildTranslationPrompt("list files in the current directory")
	if !strings.Contains(prompt, "list files in the current directory") {
		t.Fatalf("expected user request in prompt, got %q", prompt)
	}
	if !strings.Contains(prompt, "<summary>") {
		t.Fatalf("expected summary tag guidance in prompt, got %q", prompt)
	}
	if !strings.Contains(prompt, "<list-dir") {
		t.Fatalf("expected tool guidance in prompt, got %q", prompt)
	}
}

func TestLocalRunnerTranslatesPlainEnglish(t *testing.T) {
	t.Parallel()

	runner := NewLocalRunner(t.TempDir(), func(context.Context, string, string) (string, error) {
		return "<summary>done</summary>", nil
	})

	output, err := runner.Run(context.Background(), RunRequest{
		Input: "list the files in this repository",
		Model: "gemini-test",
	})
	if err != nil {
		t.Fatalf("Run returned error: %v", err)
	}
	if output != "done" {
		t.Fatalf("unexpected output: %q", output)
	}
}
