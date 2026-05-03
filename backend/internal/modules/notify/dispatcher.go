package notify

import (
	"context"
	"fmt"
	"log/slog"
	"sync"

	"github.com/VDV001/estimate-pro/backend/internal/modules/notify/domain"
	"github.com/VDV001/estimate-pro/backend/internal/modules/notify/usecase"
)

var eventMeta = map[string]struct {
	EventType domain.EventType
	Title     string
	MessageFn func(name string) string
}{
	"member.added":          {domain.EventMemberAdded, "member.added", func(name string) string { return fmt.Sprintf("%s added to project", name) }},
	"document.uploaded":     {domain.EventDocumentUploaded, "document.uploaded", func(name string) string { return fmt.Sprintf("%s uploaded a document", name) }},
	"estimation.submitted":  {domain.EventEstimationSubmitted, "estimation.submitted", func(name string) string { return fmt.Sprintf("%s submitted an estimation", name) }},
	"estimation.aggregated": {domain.EventEstimationAggregated, "estimation.aggregated", func(name string) string { return fmt.Sprintf("Estimation aggregated by %s", name) }},
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
	if !ok {
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
// Falls back to the raw user ID when the name lookup misses, matching the
// behaviour of the async HandleEvent path.
func (d *Dispatcher) RequestEstimation(ctx context.Context, projectID, userID, taskName string) error {
	name := userID
	if n, err := d.lookup.GetName(ctx, userID); err == nil {
		name = n
	}

	return d.uc.Dispatch(ctx, domain.NotifyEvent{
		EventType: domain.EventEstimationRequested,
		ProjectID: projectID,
		ActorID:   userID,
		TaskName:  taskName,
		Title:     string(domain.EventEstimationRequested),
		Message:   fmt.Sprintf("%s requested estimation for task %s", name, taskName),
	})
}

// Shutdown waits for all in-flight dispatches to complete.
func (d *Dispatcher) Shutdown() {
	d.wg.Wait()
}
