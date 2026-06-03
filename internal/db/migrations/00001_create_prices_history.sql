-- +goose Up
CREATE TABLE IF NOT EXISTS prices_history (
    id          BIGSERIAL PRIMARY KEY,
    currency    TEXT              NOT NULL,
    value       DOUBLE PRECISION  NOT NULL,
    recorded_at TIMESTAMPTZ       NOT NULL
);

CREATE INDEX IF NOT EXISTS idx_prices_history_currency_at
    ON prices_history (currency, recorded_at);

-- +goose Down
DROP INDEX IF EXISTS idx_prices_history_currency_at;
DROP TABLE IF EXISTS prices_history;
