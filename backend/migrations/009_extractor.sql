-- +goose Up

-- extractions: aggregate root for an LLM-driven task extraction run.
-- status mirrors domain.ExtractionStatus (pending/processing/completed/
-- failed/cancelled); enforced via CHECK rather than a Postgres ENUM
-- type to keep schema migrations cheap when new statuses appear.
CREATE TABLE extractions (
    id UUID PRIMARY KEY DEFAULT uuid_generate_v4(),
    document_id UUID NOT NULL REFERENCES documents(id) ON DELETE CASCADE,
    document_version_id UUID NOT NULL REFERENCES document_versions(id) ON DELETE CASCADE,
    status VARCHAR(20) NOT NULL DEFAULT 'pending'
        CHECK (status IN ('pending', 'processing', 'completed', 'failed', 'cancelled')),
    failure_reason TEXT NOT NULL DEFAULT '',
    created_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    updated_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    started_at TIMESTAMPTZ,
    completed_at TIMESTAMPTZ
);

-- Idempotency guard from ADR-016: at most one active or completed
-- extraction per (document_id, document_version_id). Failed and
-- cancelled extractions are excluded so retry / re-request remain
-- possible after terminal failure.
CREATE UNIQUE INDEX idx_extractions_active_unique
    ON extractions (document_id, document_version_id)
    WHERE status IN ('pending', 'processing', 'completed');

CREATE INDEX idx_extractions_document ON extractions (document_id);
CREATE INDEX idx_extractions_status_updated ON extractions (status, updated_at DESC);

-- extraction_events: append-only audit trail. Repository writes one
-- row per state transition in the same transaction as the
-- extractions update so history reconstruction does not depend on
-- application-level retries.
CREATE TABLE extraction_events (
    id UUID PRIMARY KEY DEFAULT uuid_generate_v4(),
    extraction_id UUID NOT NULL REFERENCES extractions(id) ON DELETE CASCADE,
    from_status VARCHAR(20) NOT NULL
        CHECK (from_status IN ('pending', 'processing', 'completed', 'failed', 'cancelled')),
    to_status VARCHAR(20) NOT NULL
        CHECK (to_status IN ('pending', 'processing', 'completed', 'failed', 'cancelled')),
    error_message TEXT NOT NULL DEFAULT '',
    actor VARCHAR(100) NOT NULL,
    created_at TIMESTAMPTZ NOT NULL DEFAULT NOW()
);

CREATE INDEX idx_extraction_events_extraction
    ON extraction_events (extraction_id, created_at);

-- extracted_tasks: the LLM output for a completed extraction. Stored
-- as a flat list rather than JSONB so downstream UIs can paginate
-- and edit individual rows without JSON surgery.
CREATE TABLE extracted_tasks (
    id UUID PRIMARY KEY DEFAULT uuid_generate_v4(),
    extraction_id UUID NOT NULL REFERENCES extractions(id) ON DELETE CASCADE,
    name VARCHAR(255) NOT NULL,
    estimate_hint VARCHAR(255) NOT NULL DEFAULT '',
    ordinal INT NOT NULL
);

CREATE INDEX idx_extracted_tasks_extraction
    ON extracted_tasks (extraction_id, ordinal);

-- +goose Down

DROP TABLE IF EXISTS extracted_tasks;
DROP TABLE IF EXISTS extraction_events;
DROP TABLE IF EXISTS extractions;
