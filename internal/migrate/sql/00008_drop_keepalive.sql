-- +goose Up
DROP TABLE IF EXISTS db_keep_alive;

-- +goose Down
SELECT 1;
