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

// Shutdown waits for all in-flight dispatches to complete.
func (d *Dispatcher) Shutdown() {
	d.wg.Wait()
}
