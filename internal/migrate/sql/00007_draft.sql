-- +goose Up
-- +goose StatementBegin
DO $$
BEGIN
    IF NOT EXISTS (
        SELECT 1 FROM information_schema.columns
        WHERE table_schema = 'public' AND table_name = 'bot' AND column_name = 'draft'
    ) THEN
        ALTER TABLE bot ADD COLUMN draft text;
    END IF;
    IF NOT EXISTS (
        SELECT 1 FROM information_schema.columns
        WHERE table_schema = 'public' AND table_name = 'bot' AND column_name = 'draftMsg'
    ) THEN
        ALTER TABLE bot ADD COLUMN "draftMsg" integer;
    END IF;
END $$;
-- +goose StatementEnd

-- +goose Down
SELECT 1;
