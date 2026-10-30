-- +goose Up
-- +goose StatementBegin
DO $$
BEGIN
    IF NOT EXISTS (
        SELECT 1 FROM information_schema.columns
        WHERE table_schema = 'public' AND table_name = 'user' AND column_name = 'notifications'
    ) THEN
        ALTER TABLE "user" ADD COLUMN notifications boolean NOT NULL DEFAULT true;
    END IF;
    IF NOT EXISTS (
        SELECT 1 FROM information_schema.columns
        WHERE table_schema = 'public' AND table_name = 'user' AND column_name = 'locations'
    ) THEN
        ALTER TABLE "user" ADD COLUMN locations character varying[] NOT NULL DEFAULT '{}';
    END IF;
END $$;
-- +goose StatementEnd

CREATE TABLE IF NOT EXISTS db_keep_alive (
    id SERIAL PRIMARY KEY,
    payload varchar(64) NOT NULL,
    "createdAt" timestamp NOT NULL DEFAULT now()
);

-- +goose Down
SELECT 1;
