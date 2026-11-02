-- +goose Up
-- +goose StatementBegin
DO $$
BEGIN
    IF NOT EXISTS (
        SELECT 1 FROM information_schema.columns
        WHERE table_schema = 'public' AND table_name = 'bot' AND column_name = 'waitingName'
    ) THEN
        ALTER TABLE bot ADD COLUMN "waitingName" varchar(128);
    END IF;
END $$;
-- +goose StatementEnd

-- +goose Down
SELECT 1;
