-- +goose Up
DROP TABLE IF EXISTS delivery_recipient;
DROP TABLE IF EXISTS delivery;

-- +goose Down
SELECT 1;
