-- +goose Up

ALTER TABLE users ADD COLUMN notification_email VARCHAR(255);

-- +goose Down

ALTER TABLE users DROP COLUMN IF EXISTS notification_email;
