package gemini

import "testing"

func TestConversationCloseKeepsDescriptionChannelUsable(t *testing.T) {
	t.Parallel()

	conv := NewConversation(nil)
	conv.Close()

	select {
	case conv.DescriptionChannel <- "still-open":
	default:
		t.Fatal("expected description channel to remain writable after Close")
	}

	select {
	case desc := <-conv.DescriptionChannel:
		if desc != "still-open" {
			t.Fatalf("unexpected description %q", desc)
		}
	default:
		t.Fatal("expected queued description to be readable")
	}
}
