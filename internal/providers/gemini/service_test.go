package gemini

import "testing"

func TestSummarizeGeminiErrorQuota(t *testing.T) {
	t.Parallel()

	got := summarizeGeminiError("Conversation title was not updated", errString("Error 429 RESOURCE_EXHAUSTED quota exceeded"))
	want := "Conversation title was not updated: Gemini API quota exceeded. Try again shortly."
	if got != want {
		t.Fatalf("unexpected summary: %q", got)
	}
}

type errString string

func (e errString) Error() string { return string(e) }
