package ui

import (
	"strings"
	"testing"
)

func TestSummarizeUserErrorQuota(t *testing.T) {
	t.Parallel()

	got := summarizeUserError(errString("Error 429 RESOURCE_EXHAUSTED quota exceeded"))
	want := "Gemini API quota exceeded. Try again shortly."
	if got != want {
		t.Fatalf("unexpected summary: %q", got)
	}
}

func TestSummarizeUserErrorPreservesWrappedProviderMessage(t *testing.T) {
	t.Parallel()

	err := errString("send message: Error 429 RESOURCE_EXHAUSTED quota exceeded")
	got := summarizeUserError(err)
	if !strings.Contains(got, "Gemini API quota exceeded") {
		t.Fatalf("expected quota summary, got %q", got)
	}
}

func TestLooksLikePromptWeaverInput(t *testing.T) {
	t.Parallel()

	if !looksLikePromptWeaverInput("<summary>done</summary>") {
		t.Fatal("expected structured PromptWeaver input to be detected")
	}
	if looksLikePromptWeaverInput("list the files in this repo") {
		t.Fatal("expected plain English input not to be detected as PromptWeaver")
	}
}

func TestBuildAgentTranslationPromptIncludesUserRequestAndTags(t *testing.T) {
	t.Parallel()

	prompt := buildAgentTranslationPrompt("list files in the current directory")
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

type errString string

func (e errString) Error() string { return string(e) }
