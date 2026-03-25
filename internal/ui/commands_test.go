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

type errString string

func (e errString) Error() string { return string(e) }
