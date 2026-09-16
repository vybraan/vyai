package gemini

import (
	"context"
	"errors"
	"net/http"
	"strings"
	"testing"
	"time"
	"unicode/utf8"

	"github.com/vybraan/vyai/internal/appconfig"
	"google.golang.org/api/googleapi"
	"google.golang.org/genproto/googleapis/rpc/errdetails"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/status"
	"google.golang.org/protobuf/types/known/durationpb"
)

func titleTestService(t *testing.T) (*GeminiService, *Conversation) {
	t.Helper()
	gs := NewGeminiService(NewConversationManager(), &appconfig.Config{DataDir: t.TempDir(), DescriptionModel: "test", DescriptionPrompt: "Title only"})
	messages := []Message{{Role: "user", Text: "Diagnose slow PostgreSQL queries"}}
	conv := gs.cm.StartNewConversation(NewPersistentHistoryRepository(messages, nil, nil))
	gs.queueTitle(conv, messages)
	return gs, conv
}

func runTitle(gs *GeminiService) {
	gs.titleMu.Lock()
	gs.idleSince = time.Time{}
	gs.titleMu.Unlock()
	gs.refineNextTitle(context.Background(), time.Now())
}

func TestTitleRetrySurvivesRestart(t *testing.T) {
	gs, conv := titleTestService(t)
	if got := conv.GetDescription(); got != "Diagnose slow PostgreSQL queries" {
		t.Fatal(got)
	}
	gs.generateTitle = func(context.Context, string, string) (string, error) { return "", errors.New("quota exceeded") }
	runTitle(gs)
	records, err := gs.store.LoadAll()
	if err != nil || len(records) != 1 {
		t.Fatalf("records: %v, %v", records, err)
	}
	if !records[0].Title.Pending || records[0].Title.Attempts != 1 || !records[0].Title.NextAttempt.After(time.Now()) {
		t.Fatalf("retry not persisted: %+v", records[0].Title)
	}
	restarted := NewGeminiService(NewConversationManager(), gs.cfg)
	if err := restarted.LoadStoredConversations(); err != nil {
		t.Fatal(err)
	}
	conv = restarted.cm.All()[0]
	called := false
	restarted.generateTitle = func(_ context.Context, _, prompt string) (string, error) {
		called = true
		if !strings.Contains(prompt, "PostgreSQL") {
			t.Fatal("lost original topic")
		}
		return "PostgreSQL query performance", nil
	}
	runTitle(restarted)
	if called {
		t.Fatal("retried before backoff elapsed")
	}
	conv.mu.Lock()
	conv.title.NextAttempt = time.Time{}
	conv.mu.Unlock()
	runTitle(restarted)
	if conv.GetDescription() != "PostgreSQL query performance" {
		t.Fatal("title not refined")
	}
	records, err = restarted.store.LoadAll()
	if err != nil || records[0].Title.Pending {
		t.Fatalf("completion not persisted: %v", err)
	}
}

func TestTitlePreemptedByForeground(t *testing.T) {
	gs, conv := titleTestService(t)
	started, exited := make(chan struct{}), make(chan struct{})
	gs.generateTitle = func(ctx context.Context, _, _ string) (string, error) {
		close(started)
		<-ctx.Done()
		close(exited)
		return "", ctx.Err()
	}
	done := make(chan struct{})
	go func() { defer close(done); runTitle(gs) }()
	select {
	case <-started:
	case <-time.After(time.Second):
		t.Fatal("worker did not start")
	}
	ctx, cancel := context.WithTimeout(context.Background(), time.Second)
	defer cancel()
	release, err := gs.BeginForeground(ctx)
	if err != nil {
		t.Fatal(err)
	}
	defer release()
	select {
	case <-exited:
	default:
		t.Fatal("foreground overlapped title request")
	}
	<-done
	conv.mu.RLock()
	pending, attempts := conv.title.Pending, conv.title.Attempts
	conv.mu.RUnlock()
	if !pending || attempts != 0 {
		t.Fatal("preemption lost or delayed the job")
	}
	gs.generateTitle = func(context.Context, string, string) (string, error) {
		t.Fatal("title ran during chat")
		return "", nil
	}
	runTitle(gs)
}

func TestTitleResultCannotOverwriteRenameOrRestoreDeletedConversation(t *testing.T) {
	for _, action := range []string{"rename", "delete"} {
		t.Run(action, func(t *testing.T) {
			gs, conv := titleTestService(t)
			gs.generateTitle = func(context.Context, string, string) (string, error) {
				var err error
				if action == "rename" {
					err = gs.RenameConversation(conv.ID, "My title")
				} else {
					err = gs.DeleteConversation(conv.ID)
				}
				if err != nil {
					t.Fatal(err)
				}
				return "Generated title", nil
			}
			runTitle(gs)
			records, err := gs.store.LoadAll()
			if err != nil {
				t.Fatal(err)
			}
			if action == "delete" && len(records) != 0 {
				t.Fatal("deleted conversation restored")
			}
			if action == "rename" && (len(records) != 1 || records[0].Description != "My title" || records[0].Title.Pending) {
				t.Fatal("manual title overwritten")
			}
		})
	}
}

func TestLegacyLockedPlaceholderIsRecovered(t *testing.T) {
	gs, conv := titleTestService(t)
	conv.mu.Lock()
	conv.description, conv.descriptionLocked, conv.title = "New Conversation...", true, TitleState{}
	conv.mu.Unlock()
	gs.persistConversation(conv)
	restarted := NewGeminiService(NewConversationManager(), gs.cfg)
	if err := restarted.LoadStoredConversations(); err != nil {
		t.Fatal(err)
	}
	loaded := restarted.cm.All()[0]
	if loaded.GetDescription() == "New Conversation..." || loaded.IsDescriptionLocked() || !loaded.title.Pending {
		t.Fatal("legacy title not recovered")
	}
}

func TestInvalidTitleStaysPending(t *testing.T) {
	for _, value := range []string{"", "\n", "New Conversation...", "two\nlines", strings.Repeat("x", 61), "escape\x1b[0m"} {
		gs, conv := titleTestService(t)
		gs.generateTitle = func(context.Context, string, string) (string, error) { return value, nil }
		runTitle(gs)
		if !conv.title.Pending || conv.title.Attempts != 1 {
			t.Fatalf("accepted invalid title: %q", value)
		}
	}
}

func TestTitleRetryHonorsProviderDelay(t *testing.T) {
	err := &googleapi.Error{Code: 429, Header: http.Header{"Retry-After": []string{"3600"}}}
	if titleRetryDelay(err, 1) < time.Hour {
		t.Fatal("ignored Retry-After")
	}
	s, e := status.New(codes.ResourceExhausted, "quota").WithDetails(&errdetails.RetryInfo{RetryDelay: durationpb.New(time.Hour)})
	if e != nil {
		t.Fatal(e)
	}
	if titleRetryDelay(s.Err(), 1) < time.Hour {
		t.Fatal("ignored RetryInfo")
	}
	for _, attempt := range []int{1, 20} {
		delay := titleRetryDelay(nil, attempt)
		if delay < 30*time.Second || delay > 38*time.Minute {
			t.Fatal(delay)
		}
	}
}

func TestTitleQuotaPausesWholeQueueAfterRestart(t *testing.T) {
	gs, _ := titleTestService(t)
	gs.generateTitle = func(context.Context, string, string) (string, error) {
		return "", &googleapi.Error{Code: 429, Header: http.Header{"Retry-After": []string{"3600"}}}
	}
	runTitle(gs)
	messages := []Message{{Role: "user", Text: "Another topic"}}
	other := gs.cm.StartNewConversation(NewPersistentHistoryRepository(messages, nil, nil))
	gs.queueTitle(other, messages)
	restarted := NewGeminiService(NewConversationManager(), gs.cfg)
	if err := restarted.LoadStoredConversations(); err != nil {
		t.Fatal(err)
	}
	restarted.generateTitle = func(context.Context, string, string) (string, error) {
		t.Fatal("next title bypassed provider cooldown")
		return "", nil
	}
	runTitle(restarted)
}

func TestLocalTitleUnicode(t *testing.T) {
	title := localTitle(strings.Repeat("界", 80))
	if !utf8.ValidString(title) || len([]rune(title)) > 60 {
		t.Fatal("invalid title truncation")
	}
}
