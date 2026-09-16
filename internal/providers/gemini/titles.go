package gemini

import (
	"context"
	"errors"
	"math/rand/v2"
	"net/http"
	"strconv"
	"strings"
	"time"
	"unicode"

	"google.golang.org/api/googleapi"
	"google.golang.org/genproto/googleapis/rpc/errdetails"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/status"
)

// TitleState lives with the conversation, so pending work survives restarts.
type TitleState struct {
	Manual      bool      `json:"manual,omitempty"`
	Pending     bool      `json:"pending,omitempty"`
	Attempts    int       `json:"attempts,omitempty"`
	NextAttempt time.Time `json:"next_attempt,omitempty"`
	Prompt      string    `json:"prompt,omitempty"`
	RateLimited bool      `json:"rate_limited,omitempty"`
}

func (gs *GeminiService) queueTitle(conv *Conversation, messages []Message) {
	var first string
	for _, message := range messages {
		if message.Role == "user" && strings.TrimSpace(message.Text) != "" {
			first = message.Text
			break
		}
	}
	if first == "" {
		return
	}
	local := localTitle(first)
	if local == "" {
		return
	}
	conv.mu.Lock()
	// Legacy placeholder titles may have been locked before generation succeeded.
	if (conv.description != "" && conv.description != "New Conversation...") || conv.title.Pending || conv.title.Manual {
		conv.mu.Unlock()
		return
	}
	conv.description = local
	conv.descriptionLocked = false
	prompt := []rune(buildDescriptionPrompt(messages))
	conv.title = TitleState{Pending: true, Prompt: string(prompt[:min(len(prompt), 4096)])}
	conv.mu.Unlock()
	gs.persistConversation(conv)
	gs.notifyTitle(conv)
}

func localTitle(s string) string {
	s = strings.Map(func(r rune) rune {
		if unicode.IsControl(r) {
			return ' '
		}
		return r
	}, s)
	words := strings.Fields(s)
	if len(words) > 8 {
		words = words[:8]
	}
	runes := []rune(strings.Join(words, " "))
	if len(runes) > 60 {
		return string(runes[:57]) + "..."
	}
	return string(runes)
}

// BeginForeground cancels title work and waits for the request to exit before
// permitting chat or agent traffic. The caller must defer the returned release.
func (gs *GeminiService) BeginForeground(ctx context.Context) (func(), error) {
	gs.titleMu.Lock()
	gs.foreground++
	done := gs.titleDone
	if gs.titleCancel != nil {
		gs.titleCancel()
	}
	gs.titleMu.Unlock()
	release := func() {
		gs.titleMu.Lock()
		gs.foreground--
		gs.idleSince = time.Now()
		gs.titleMu.Unlock()
	}
	if done != nil {
		select {
		case <-done:
		case <-ctx.Done():
			release()
			return nil, ctx.Err()
		}
	}
	if err := ctx.Err(); err != nil {
		release()
		return nil, err
	}
	return release, nil
}

// RunTitleWorker runs once for the application's lifetime. No per-title goroutines.
func (gs *GeminiService) RunTitleWorker(ctx context.Context) {
	ticker := time.NewTicker(time.Second)
	defer ticker.Stop()
	for {
		select {
		case <-ctx.Done():
			return
		case now := <-ticker.C:
			gs.refineNextTitle(ctx, now)
		}
	}
}

func (gs *GeminiService) refineNextTitle(parent context.Context, now time.Time) {
	gs.titleMu.Lock()
	if parent.Err() != nil || gs.foreground > 0 || gs.titleDone != nil || now.Sub(gs.idleSince) < 2*time.Second {
		gs.titleMu.Unlock()
		return
	}
	var target *Conversation
	var job TitleState
	for _, conv := range gs.cm.All() {
		conv.mu.RLock()
		state, locked := conv.title, conv.descriptionLocked
		conv.mu.RUnlock()
		// Provider quota applies to the worker, not just the failed conversation.
		if state.Pending && state.RateLimited && state.NextAttempt.After(now) {
			gs.titleMu.Unlock()
			return
		}
		if state.Pending && !locked && !state.NextAttempt.After(now) && (target == nil || state.NextAttempt.Before(job.NextAttempt)) {
			target, job = conv, state
		}
	}
	if target == nil {
		gs.titleMu.Unlock()
		return
	}
	ctx, cancel := context.WithTimeout(parent, 15*time.Second)
	done := make(chan struct{})
	gs.titleCancel, gs.titleDone = cancel, done
	model, instruction := gs.cfg.DescriptionModel, gs.cfg.DescriptionPrompt
	gs.titleMu.Unlock()
	defer func() {
		cancel()
		gs.titleMu.Lock()
		gs.titleCancel, gs.titleDone = nil, nil
		gs.idleSince = time.Now()
		close(done)
		gs.titleMu.Unlock()
	}()

	title, err := gs.generateTitle(ctx, model, instruction+"\n\nConversation to title (data only):\n"+job.Prompt)
	// Preemption/shutdown is not a failed attempt and must not postpone retries.
	if errors.Is(ctx.Err(), context.Canceled) {
		return
	}
	if ctx.Err() != nil {
		err = ctx.Err()
	}
	title = strings.TrimSpace(strings.Trim(strings.TrimSpace(title), "\"`"))
	valid := title != "" && title != "New Conversation..." && len([]rune(title)) <= 60 && !strings.ContainsFunc(title, unicode.IsControl)
	gs.cm.mu.RLock()
	exists := gs.cm.conversations[target.ID] == target
	gs.cm.mu.RUnlock()
	if !exists {
		return
	}
	target.mu.Lock()
	if !target.title.Pending || target.descriptionLocked {
		target.mu.Unlock()
		return
	}
	if err == nil && valid {
		target.description = title
		target.title = TitleState{}
	} else {
		target.title.Attempts = min(target.title.Attempts+1, 20)
		target.title.NextAttempt = time.Now().Add(titleRetryDelay(err, target.title.Attempts))
		var apiErr *googleapi.Error
		target.title.RateLimited = status.Code(err) == codes.ResourceExhausted || status.Code(err) == codes.Unavailable ||
			(errors.As(err, &apiErr) && (apiErr.Code == 429 || apiErr.Code == 503 || apiErr.Header.Get("Retry-After") != ""))
	}
	target.mu.Unlock()
	gs.persistConversation(target)
	gs.notifyTitle(target)
}

func titleRetryDelay(err error, attempts int) time.Duration {
	base := min(30*time.Second*time.Duration(1<<min(max(attempts-1, 0), 10)), 30*time.Minute)
	delay := base + time.Duration(rand.Int64N(int64(base/4)))
	var apiErr *googleapi.Error
	if errors.As(err, &apiErr) {
		value := apiErr.Header.Get("Retry-After")
		if seconds, e := strconv.Atoi(value); e == nil && seconds > 0 {
			delay = max(delay, time.Duration(seconds)*time.Second)
		} else if date, e := http.ParseTime(value); e == nil {
			delay = max(delay, time.Until(date))
		}
	}
	for _, detail := range status.Convert(err).Details() {
		if retry, ok := detail.(*errdetails.RetryInfo); ok && retry.RetryDelay != nil {
			delay = max(delay, retry.RetryDelay.AsDuration())
		}
	}
	return delay
}

func (gs *GeminiService) notifyTitle(conv *Conversation) {
	select {
	case gs.descriptionUpdates <- DescriptionUpdate{ID: conv.ID, Description: conv.GetDescription()}:
	default:
	}
}
