package ui

import (
	"strings"
	"testing"

	"github.com/charmbracelet/bubbles/v2/viewport"
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

func TestStreamMessagesAppendDeltas(t *testing.T) {
	t.Parallel()

	m := UIModel{width: 80, viewport: viewport.New(viewport.WithWidth(80), viewport.WithHeight(10))}
	model, _ := m.Update(streamStartMsg{firstToken: "hello", tokens: make(chan string), errCh: make(chan error)})
	m = model.(UIModel)
	model, _ = m.Update(streamMsg(" world"))
	m = model.(UIModel)

	if m.partialResponse != "hello world" {
		t.Fatalf("unexpected partial response: %q", m.partialResponse)
	}
}

type errString string

func (e errString) Error() string { return string(e) }
