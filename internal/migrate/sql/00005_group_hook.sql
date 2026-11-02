-- +goose Up
-- +goose StatementBegin
DO $$
BEGIN
    IF NOT EXISTS (
        SELECT 1 FROM information_schema.columns
        WHERE table_schema = 'public' AND table_name = 'bot' AND column_name = 'waitingUser'
    ) THEN
        ALTER TABLE bot ADD COLUMN "waitingUser" bigint;
    END IF;
    IF NOT EXISTS (
        SELECT 1 FROM information_schema.columns
        WHERE table_schema = 'public' AND table_name = 'user' AND column_name = 'locationsEditorId'
    ) THEN
        ALTER TABLE "user" ADD COLUMN "locationsEditorId" bigint;
    END IF;
    IF NOT EXISTS (
        SELECT 1 FROM information_schema.columns
        WHERE table_schema = 'public' AND table_name = 'user' AND column_name = 'locationsEditor'
    ) THEN
        ALTER TABLE "user" ADD COLUMN "locationsEditor" varchar(128);
    END IF;
    IF NOT EXISTS (
        SELECT 1 FROM information_schema.columns
        WHERE table_schema = 'public' AND table_name = 'user' AND column_name = 'webhook_url'
    ) THEN
        ALTER TABLE "user" ADD COLUMN webhook_url varchar(2048);
    END IF;
END $$;
-- +goose StatementEnd

-- +goose Down
SELECT 1;
