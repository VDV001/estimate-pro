// Copyright 2026 Daniil Vdovin. All rights reserved.
// SPDX-License-Identifier: AGPL-3.0-only

// Package worker hosts the asynchronous body of the extractor
// pipeline. The Process method orchestrates: load extraction by
// ID, transition to processing, fetch document bytes from the
// document storage adapter, dispatch through shared/reader to
// extract plain text, run a prompt-injection guard, call the
// shared/llm completer with a JSON-mode prompt, validate the
// response schema, persist the extracted tasks, and stamp the
// completed audit event. PR-B3 builds the body slice by slice
// under TDD; this file grows commit by commit.
package worker

import (
	"context"
	"encoding/json"
	"fmt"
	"strings"
	"time"

	"github.com/VDV001/estimate-pro/backend/internal/modules/extractor/domain"
	sharedllm "github.com/VDV001/estimate-pro/backend/internal/shared/llm"
)

// ExtractionArgs is the JSON payload enqueued onto the river queue
// for a single extraction job. The shape is pinned across the
// PR-B2 / PR-B3 boundary so the queue contract stays stable.
type ExtractionArgs struct {
	ExtractionID string `json:"extraction_id"`
}

// ExtractionWorker is the river-compatible worker for processing
// extraction jobs. Dependencies are injected at construction so the
// worker can be unit-tested with panic-on-unexpected-call fakes
// and integration-tested against postgres + minio + httptest LLM.
type ExtractionWorker struct {
	store    ExtractionStore
	source   DocumentSource
	reader   TextExtractor
	llm      Completer
	security SecurityChecker
}

// NewExtractionWorker wires the worker with its five collaborators.
// All five are required — passing nil for any of them is a
// programmer error and will surface as a nil-pointer panic the
// first time the corresponding code path executes. The composition
// root in cmd/server/main.go is the single instantiation site,
// gated behind FEATURE_DOCUMENT_PIPELINE_ENABLED.
func NewExtractionWorker(store ExtractionStore, source DocumentSource, reader TextExtractor, llm Completer, security SecurityChecker) *ExtractionWorker {
	return &ExtractionWorker{
		store:    store,
		source:   source,
		reader:   reader,
		llm:      llm,
		security: security,
	}
}

// Process is invoked by the river job runner for each enqueued
// ExtractionArgs. PR-B3 builds the body slice by slice — currently
// the load + status-guard + pending->processing transition.
// Subsequent commits add the download / parse / security / LLM /
// persist stages each behind their own RED+GREEN pair.
//
// Idempotency: if the extraction has already moved past pending
// (processing / completed / failed / cancelled), Process returns
// nil without side effects. River may re-dispatch the same args
// after a worker crash, and the second invocation must observe
// the current state and exit cleanly.
//
// actor is hard-coded to "worker" — every transition driven by
// this method is the system, not a user; user-driven transitions
// flow through the Extractor use-cases with the user's identifier
// supplied by the HTTP handler.
const workerActor = "worker"

// readerTimeout caps the time spent inside the document reader
// composite — large XLSX or PDF files are the realistic worst-case
// (>1M cells / encrypted layers). The cap is per-document, not
// per-job; the surrounding river job timeout (5min) covers the
// other stages. ADR-016 §timeouts.
const readerTimeout = 10 * time.Second

// llmMaxTokens is the response budget handed to every extraction
// LLM call. 4096 fits roughly 30 tasks at ~120 tokens per task
// (name + estimate_hint + JSON syntax) which covers the realistic
// upper bound for a single ТЗ. The adapter-default of 1024 is
// silently truncating, so we set this explicitly per ADR-016.
const llmMaxTokens = 4096

// extractionSystemPrompt is the static contract handed to every
// LLM call: extract tasks, return strict JSON. The prompt is in
// Russian because the documents and the operator surface (bot,
// frontend) are Russian — keeping the LLM in the same language
// reduces translation drift in task names. JSON schema is
// validated by the worker on the response (Pair 5), so the
// prompt only needs to elicit the shape, not enforce it.
const extractionSystemPrompt = `Ты извлекаешь задачи из технического задания (ТЗ) для дальнейшей оценки.
Каждая задача — самостоятельная единица оценки (типичный размер: от часа до нескольких дней работы одного инженера).

Верни строго JSON-объект, без markdown, без комментариев, без обрамляющего текста.

Схема:
{
  "tasks": [
    {"name": "<короткое название задачи (1..255 символов)>", "estimate_hint": "<подсказка оценщику: часы, сложность, риски — может быть пустой>"}
  ]
}

Если ТЗ не содержит задач (пустой документ, нерелевантный текст), верни {"tasks": []}.`

// buildLLMPrompt assembles the user-side prompt for a single
// extraction call. The system prompt is static; the user prompt
// embeds the extracted document text verbatim so the model sees
// the original structure (line breaks, headings, lists) rather
// than a re-flowed paragraph. PR-B5+ may extend this to include
// project name + description as additional context — for now the
// extraction module does not depend on the project module and
// the document text alone is enough for MVP.
func buildLLMPrompt(text string) (system, user string) {
	return extractionSystemPrompt, "Текст ТЗ:\n\n" + text
}

func (w *ExtractionWorker) Process(ctx context.Context, args ExtractionArgs) error {
	ext, err := w.store.GetByID(ctx, args.ExtractionID)
	if err != nil {
		return fmt.Errorf("worker.Process load extraction %q: %w", args.ExtractionID, err)
	}
	if ext.Status != domain.StatusPending {
		return nil
	}
	if err := w.transition(ctx, ext, (*domain.Extraction).MarkProcessing, ""); err != nil {
		return fmt.Errorf("worker.Process transition pending->processing: %w", err)
	}

	data, filename, err := w.source.Fetch(ctx, ext.DocumentVersionID)
	if err != nil {
		return fmt.Errorf("worker.Process fetch document %q: %w", ext.DocumentVersionID, err)
	}

	parseCtx, cancel := context.WithTimeout(ctx, readerTimeout)
	defer cancel()
	text, err := w.reader.Parse(parseCtx, filename, data)
	if err != nil {
		return fmt.Errorf("worker.Process parse document %q: %w", filename, err)
	}

	if w.security.IsPromptInjection(text) {
		if err := w.markFailed(ctx, ext, promptInjectionReason); err != nil {
			return fmt.Errorf("worker.Process record prompt-injection failure: %w", err)
		}
		return fmt.Errorf("worker.Process security guard for extraction %q: %w", ext.ID, domain.ErrPromptInjectionDetected)
	}

	systemPrompt, userPrompt := buildLLMPrompt(text)
	rawResponse, _, err := w.llm.Complete(ctx, systemPrompt, userPrompt, sharedllm.CompletionOptions{
		MaxTokens: llmMaxTokens,
		JSONMode:  true,
	})
	if err != nil {
		return fmt.Errorf("worker.Process LLM dispatch for extraction %q: %w", ext.ID, err)
	}

	if _, err := parseLLMResponse(rawResponse); err != nil {
		if markErr := w.markFailed(ctx, ext, schemaInvalidReason); markErr != nil {
			return fmt.Errorf("worker.Process record schema-invalid failure: %w", markErr)
		}
		return fmt.Errorf("worker.Process LLM response invalid for extraction %q: %w", ext.ID, domain.ErrLLMResponseSchemaInvalid)
	}

	return nil
}

// schemaInvalidReason is the audit-event reason recorded when the
// LLM response fails the schema gate. Operators reading the
// extraction_events row see a uniform reason regardless of the
// specific failure mode (non-JSON / wrong shape / missing field)
// — drilling down to the exact mode goes through structured
// logging once observability lands.
const schemaInvalidReason = "LLM response failed schema validation"

// llmExtractResponse is the on-the-wire shape of the LLM's reply.
// Adapters call json.Unmarshal into this struct; the worker then
// validates per-task invariants before mapping to domain.ExtractedTask.
// Keeping the JSON layer separate from the domain VO lets us evolve
// the wire format (e.g. add a confidence field) without touching
// domain invariants.
type llmExtractResponse struct {
	Tasks []llmExtractedTask `json:"tasks"`
}

type llmExtractedTask struct {
	Name         string `json:"name"`
	EstimateHint string `json:"estimate_hint"`
}

// parseLLMResponse decodes the LLM's reply and validates the
// schema. Returns ErrLLMResponseSchemaInvalid for any of: non-JSON
// input, missing/wrong-typed 'tasks' field, item without 'name',
// item with empty/whitespace 'name'. The "tasks": [] case is
// considered valid (extraction document yielded zero tasks) — the
// caller decides whether that is a happy path or a soft failure.
func parseLLMResponse(raw string) (*llmExtractResponse, error) {
	var resp llmExtractResponse
	dec := json.NewDecoder(strings.NewReader(raw))
	dec.DisallowUnknownFields()
	if err := dec.Decode(&resp); err != nil {
		return nil, fmt.Errorf("decode JSON: %w", domain.ErrLLMResponseSchemaInvalid)
	}
	for i, task := range resp.Tasks {
		if strings.TrimSpace(task.Name) == "" {
			return nil, fmt.Errorf("task[%d].name is empty: %w", i, domain.ErrLLMResponseSchemaInvalid)
		}
	}
	return &resp, nil
}

// promptInjectionReason is the audit-event reason recorded when the
// security guard fires. Kept as a const so the failure UX, the
// audit trail, and any future operator dashboard share one source
// of truth.
const promptInjectionReason = "prompt injection detected"

// markFailed records a processing->failed transition with the
// supplied reason. The reason is stored both on the Extraction
// (via Extraction.MarkFailed) and on the audit ExtractionEvent
// (via NewExtractionEvent's errorMessage parameter) so it is
// visible to operators reading either the current row or the
// event timeline.
func (w *ExtractionWorker) markFailed(ctx context.Context, ext *domain.Extraction, reason string) error {
	return w.transition(ctx, ext, func(e *domain.Extraction) error {
		return e.MarkFailed(reason)
	}, reason)
}

// transition mutates the extraction via the supplied state-machine
// method, then records the audit event in a single UpdateStatus
// call so the post-mutation status and the audit trail are committed
// atomically by the repository. The transition method is passed as
// a Go method expression — callers write
// (*domain.Extraction).MarkProcessing rather than building a closure.
func (w *ExtractionWorker) transition(ctx context.Context, ext *domain.Extraction, mutate func(*domain.Extraction) error, errorMessage string) error {
	from := ext.Status
	if err := mutate(ext); err != nil {
		return fmt.Errorf("transition mutate from %s: %w", from, err)
	}
	event, err := domain.NewExtractionEvent(ext.ID, from, ext.Status, errorMessage, workerActor)
	if err != nil {
		return fmt.Errorf("transition build audit event %s->%s: %w", from, ext.Status, err)
	}
	if err := w.store.UpdateStatus(ctx, ext, event); err != nil {
		return fmt.Errorf("transition persist %s->%s: %w", from, ext.Status, err)
	}
	return nil
}
