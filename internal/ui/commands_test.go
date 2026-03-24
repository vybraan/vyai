package ui

import "testing"

func TestSummarizeUserErrorQuota(t *testing.T) {
	t.Parallel()

	got := summarizeUserError(errString("Error 429 RESOURCE_EXHAUSTED quota exceeded"))
	want := "Gemini API quota exceeded. Try again shortly."
	if got != want {
		t.Fatalf("unexpected summary: %q", got)
	}
}

type errString string

func (e errString) Error() string { return string(e) }
