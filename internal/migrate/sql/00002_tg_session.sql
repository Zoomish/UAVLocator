-- +goose Up
CREATE TABLE IF NOT EXISTS tg_session (
    id smallint PRIMARY KEY,
    data bytea NOT NULL,
    seed text NOT NULL DEFAULT '',
    "updatedAt" timestamp NOT NULL DEFAULT now()
);

-- +goose Down
SELECT 1;
