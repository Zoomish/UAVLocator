-- +goose Up
-- +goose StatementBegin
DO $$
BEGIN
    IF NOT EXISTS (
        SELECT 1 FROM information_schema.columns
        WHERE table_schema = 'public' AND table_name = 'user' AND column_name = 'blocked'
    ) THEN
        ALTER TABLE "user" ADD COLUMN blocked boolean NOT NULL DEFAULT false;
    END IF;
END $$;
-- +goose StatementEnd

-- +goose Down
SELECT 1;
