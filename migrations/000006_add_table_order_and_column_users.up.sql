-- migrations/001_create_orders.sql

CREATE TYPE tier_user AS ENUM (
    'free',
    'premium'
);

ALTER TABLE users ADD COLUMN tier tier_user DEFAULT 'free';

CREATE EXTENSION IF NOT EXISTS "pgcrypto";

CREATE TABLE orders (
    id              UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    order_id        VARCHAR(50) UNIQUE NOT NULL,
    user_id         UUID NOT NULL,

    plan            VARCHAR(20) NOT NULL DEFAULT 'premium',
    gross_amount    BIGINT NOT NULL,

    gateway         VARCHAR(20) NOT NULL DEFAULT 'midtrans',
    gateway_tx_id   VARCHAR(100),

    status          VARCHAR(20) NOT NULL DEFAULT 'pending' 
                    CHECK (status IN ('pending', 'settled', 'expired', 'cancel', 'deny', 'refund')),

    snap_token      VARCHAR(255),
    webhook_payload JSONb,

    expired_at      TIMESTAMPTZ NOT NULL,
    paid_at         TIMESTAMPTZ,
    created_at      TIMESTAMPTZ NOT NULL DEFAULT now(),
    updated_at      TIMESTAMPTZ NOT NULL DEFAULT now()
);

CREATE INDEX idx_orders_user_id ON orders(user_id, created_at DESC);
CREATE INDEX idx_orders_status ON orders(status);