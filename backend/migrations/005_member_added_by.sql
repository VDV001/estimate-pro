-- +goose Up

ALTER TABLE project_members ADD COLUMN added_by UUID REFERENCES users(id);
ALTER TABLE project_members ADD COLUMN added_at TIMESTAMPTZ NOT NULL DEFAULT NOW();

-- +goose Down

ALTER TABLE project_members DROP COLUMN IF EXISTS added_at;
ALTER TABLE project_members DROP COLUMN IF EXISTS added_by;
