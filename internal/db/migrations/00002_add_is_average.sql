-- +goose Up
ALTER TABLE prices_history ADD COLUMN is_average BOOLEAN NOT NULL DEFAULT FALSE;

-- +goose Down
ALTER TABLE prices_history DROP COLUMN is_average;
