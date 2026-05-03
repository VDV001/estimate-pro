package notify

import (
	"context"
	"fmt"
	"log/slog"
	"sync"

	"github.com/VDV001/estimate-pro/backend/internal/modules/notify/domain"
	"github.com/VDV001/estimate-pro/backend/internal/modules/notify/usecase"
)

// eventMeta describes how to render a notification for each domain event.
//
// MessageFn accepts variadic extra args so events that need additional context
// (e.g. task name for estimation.requested) can pass it without forcing every
// existing event to deal with the extra parameter.
//
// SyncOnly marks events that must only flow through a typed sync entry point
// (e.g. Dispatcher.RequestEstimation) — HandleEvent's generic async path
// rejects them, because the caller has no way to supply the extra args and
// MessageFn would silently render a half-formed message.
var eventMeta = map[string]struct {
	EventType domain.EventType
	Title     string
	SyncOnly  bool
	MessageFn func(name string, args ...string) string
}{
	"member.added":          {domain.EventMemberAdded, "member.added", false, func(name string, _ ...string) string { return fmt.Sprintf("%s added to project", name) }},
	"document.uploaded":     {domain.EventDocumentUploaded, "document.uploaded", false, func(name string, _ ...string) string { return fmt.Sprintf("%s uploaded a document", name) }},
	"estimation.submitted":  {domain.EventEstimationSubmitted, "estimation.submitted", false, func(name string, _ ...string) string { return fmt.Sprintf("%s submitted an estimation", name) }},
	"estimation.aggregated": {domain.EventEstimationAggregated, "estimation.aggregated", false, func(name string, _ ...string) string { return fmt.Sprintf("Estimation aggregated by %s", name) }},
	"estimation.requested": {domain.EventEstimationRequested, "estimation.requested", true, func(name string, args ...string) string {
		var taskName string
		if len(args) > 0 {
			taskName = args[0]
		}
		return fmt.Sprintf("%s requested estimation for task %s", name, taskName)
	}},
}

// UserNameLookup resolves user ID to display name.
type UserNameLookup interface {
	GetName(ctx context.Context, userID string) (string, error)
}

// Dispatcher receives events and triggers notification creation.
type Dispatcher struct {
	uc     *usecase.NotificationUsecase
	lookup UserNameLookup
	ctx    context.Context
	wg     sync.WaitGroup
}

func NewDispatcher(uc *usecase.NotificationUsecase, lookup UserNameLookup, ctx context.Context) *Dispatcher {
	return &Dispatcher{uc: uc, lookup: lookup, ctx: ctx}
}

// HandleEvent is the callback for emitEvent. Runs notification dispatch in background goroutine.
func (d *Dispatcher) HandleEvent(eventType, projectID, userID string) {
	meta, ok := eventMeta[eventType]
	if !ok || meta.SyncOnly {
		return
	}

	d.wg.Add(1)
	go func() {
		defer d.wg.Done()

		name := userID
		if n, err := d.lookup.GetName(d.ctx, userID); err == nil {
			name = n
		}

		err := d.uc.Dispatch(d.ctx, domain.NotifyEvent{
			EventType: meta.EventType,
			ProjectID: projectID,
			ActorID:   userID,
			Title:     meta.Title,
			Message:   meta.MessageFn(name),
		})
		if err != nil {
			slog.Error("notification dispatch failed", "event", eventType, "project", projectID, "error", err)
		}
	}()
}

// RequestEstimation synchronously dispatches an EventEstimationRequested
// notification to every project member except the actor. Used by the bot
// `request_estimation` intent — caller blocks on the result so a failed
// dispatch surfaces to the user instead of being silently fire-and-forget.
//
// Title and message templates come from eventMeta — same source of truth as
// the async HandleEvent path. Falls back to the raw user ID when the name
// lookup misses, matching dispatcher.go:52-55.
func (d *Dispatcher) RequestEstimation(ctx context.Context, projectID, userID, taskName string) error {
	meta := eventMeta[string(domain.EventEstimationRequested)]

	name := userID
	if n, err := d.lookup.GetName(ctx, userID); err == nil {
		name = n
	}

	return d.uc.Dispatch(ctx, domain.NotifyEvent{
		EventType: meta.EventType,
		ProjectID: projectID,
		ActorID:   userID,
		Title:     meta.Title,
		Message:   meta.MessageFn(name, taskName),
	})
}

// Shutdown waits for all in-flight dispatches to complete.
func (d *Dispatcher) Shutdown() {
	d.wg.Wait()
}
