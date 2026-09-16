package ui

import (
	"strings"
	"testing"

	"github.com/charmbracelet/bubbles/v2/viewport"
	tea "github.com/charmbracelet/bubbletea/v2"
	"github.com/charmbracelet/lipgloss/v2"
	"github.com/vybraan/vyai/internal/appconfig"
	"github.com/vybraan/vyai/internal/providers/gemini"
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

func chatTestModel(width, height int) UIModel {
	gs := gemini.NewGeminiService(gemini.NewConversationManager(), &appconfig.Config{ChatModel: "gemini-test"})
	m := NewUIModel(gs, "", nil)
	model, _ := m.Update(tea.WindowSizeMsg{Width: width, Height: height})
	return model.(UIModel)
}

func TestStreamingLayoutAndMouse(t *testing.T) {
	for _, width := range []int{40, 80, 120} {
		m := chatTestModel(width, 24)
		m.state = Insert
		if h := lipgloss.Height(m.View()); h != 24 {
			t.Fatalf("idle height = %d", h)
		}
		m.loading = true
		if h := lipgloss.Height(m.View()); h != 24 {
			t.Fatalf("waiting height = %d", h)
		}
		m.messages = []string{renderUserMessage(strings.Repeat("history\n", 40))}
		m.renderViewport(strings.Join(m.messages, "\n"))
		m.viewport.GotoBottom()
		model, _ := m.Update(streamStartMsg{firstToken: "hello", tokens: make(chan string), errCh: make(chan error)})
		m = model.(UIModel)
		model, _ = m.Update(tea.MouseWheelMsg{Button: tea.MouseWheelUp})
		m = model.(UIModel)
		if m.viewport.AtBottom() {
			t.Fatal("mouse wheel did not scroll while input was focused")
		}
		offset := m.viewport.YOffset
		model, _ = m.Update(streamMsg(strings.Repeat(" next line\n", 30)))
		m = model.(UIModel)
		if m.viewport.YOffset != offset {
			t.Fatal("stream forced the viewport back to the bottom")
		}
		for _, notice := range []string{"", strings.Repeat("service notice ", 15)} {
			model, _ = m.Update(serviceNoticeMsg(notice))
			m = model.(UIModel)
			view := m.View()
			if h := lipgloss.Height(view); h != 24 {
				t.Fatalf("width %d: frame height = %d, want 24", width, h)
			}
			if w := lipgloss.Width(view); w > width {
				t.Fatalf("frame width = %d, exceeds %d", w, width)
			}
			if !strings.Contains(view, "UTF-8") {
				t.Fatal("footer missing")
			}
		}
		model, _ = m.Update(streamEndMsg{})
		m = model.(UIModel)
		if m.viewport.YOffset != offset {
			t.Fatal("completion changed the scroll position")
		}
	}
}

func TestPollStreamDrainsQueuedChunksBeforeEnd(t *testing.T) {
	tokens := make(chan string, 3)
	for _, chunk := range []string{"one", " two", " three"} {
		tokens <- chunk
	}
	close(tokens)
	m := UIModel{streamTokens: tokens, streamErr: make(chan error, 1)}
	if got := pollStreamCmd(m)(); got != streamMsg("one two three") {
		t.Fatalf("lost queued text: %#v", got)
	}
	if _, ok := pollStreamCmd(m)().(streamEndMsg); !ok {
		t.Fatal("missing stream completion")
	}
}

func BenchmarkStreamingRedraw(b *testing.B) {
	m := chatTestModel(120, 40)
	m.loading, m.streaming = true, true
	m.messages = []string{renderUserMessage(strings.Repeat("previous conversation\n", 1000))}
	m.streamHistory = m.formatViewport(strings.Join(m.messages, "\n"))
	m.partialResponse = strings.Repeat("generated response text\n", 100)
	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		m.renderStream()
		_ = m.View()
	}
}

func (e errString) Error() string { return string(e) }
