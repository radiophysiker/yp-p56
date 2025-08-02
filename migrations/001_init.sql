-- +goose Up
-- +goose StatementBegin
CREATE TYPE order_status AS ENUM ('NEW', 'PROCESSING', 'INVALID', 'PROCESSED');

CREATE TABLE IF NOT EXISTS users (
    id               UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    login            TEXT NOT NULL UNIQUE,
    password_hash    TEXT NOT NULL,
    current_balance  NUMERIC(20,2) NOT NULL DEFAULT 0,
    withdrawn_balance NUMERIC(20,2) NOT NULL DEFAULT 0,
    created_at       TIMESTAMPTZ NOT NULL DEFAULT NOW()
);

CREATE TABLE IF NOT EXISTS orders (
    id            BIGSERIAL PRIMARY KEY,
    user_id       UUID NOT NULL REFERENCES users(id) ON DELETE CASCADE,
    number        TEXT NOT NULL UNIQUE,
    status        order_status NOT NULL DEFAULT 'NEW',
    accrual       NUMERIC(20,2) NULL,
    uploaded_at   TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    processed_at  TIMESTAMPTZ NULL,
    CONSTRAINT accrual_non_negative CHECK (accrual IS NULL OR accrual >= 0)
);

CREATE INDEX IF NOT EXISTS idx_orders_user_id ON orders(user_id);

CREATE TABLE IF NOT EXISTS withdrawals (
    id            BIGSERIAL PRIMARY KEY,
    user_id       UUID NOT NULL REFERENCES users(id) ON DELETE CASCADE,
    order_number  TEXT NOT NULL,
    amount        NUMERIC(20,2) NOT NULL,
    processed_at  TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    CONSTRAINT amount_positive CHECK (amount > 0)
);

CREATE INDEX IF NOT EXISTS idx_withdrawals_user_id ON withdrawals(user_id);
-- +goose StatementEnd

-- +goose Down
-- +goose StatementBegin
DROP INDEX IF EXISTS idx_withdrawals_user_id;
DROP TABLE IF EXISTS withdrawals;

DROP INDEX IF EXISTS idx_orders_user_id;
DROP TABLE IF EXISTS orders;

DROP TABLE IF EXISTS users;

DROP TYPE IF EXISTS order_status;
-- +goose StatementEnd
