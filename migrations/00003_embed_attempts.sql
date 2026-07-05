-- +goose Up
ALTER TABLE memories
    ADD COLUMN embed_attempts        int NOT NULL DEFAULT 0,
    ADD COLUMN embed_last_attempt_at timestamptz;

ALTER TABLE note_chunks
    ADD COLUMN embed_attempts        int NOT NULL DEFAULT 0,
    ADD COLUMN embed_last_attempt_at timestamptz;

-- +goose Down
ALTER TABLE note_chunks
    DROP COLUMN embed_last_attempt_at,
    DROP COLUMN embed_attempts;

ALTER TABLE memories
    DROP COLUMN embed_last_attempt_at,
    DROP COLUMN embed_attempts;
