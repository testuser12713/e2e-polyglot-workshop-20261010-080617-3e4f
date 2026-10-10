-- Initial schema for the workshop customer portal.
-- All money is stored as integer cents. No SQLite anywhere: this runs on
-- PostgreSQL 18.

CREATE TABLE IF NOT EXISTS customers (
    id         BIGINT GENERATED ALWAYS AS IDENTITY PRIMARY KEY,
    name       TEXT        NOT NULL,
    email      TEXT        NOT NULL,
    phone      TEXT        NOT NULL,
    created_at TIMESTAMPTZ NOT NULL DEFAULT now()
);

CREATE TABLE IF NOT EXISTS vehicles (
    id            BIGINT GENERATED ALWAYS AS IDENTITY PRIMARY KEY,
    license_plate TEXT        NOT NULL UNIQUE,
    make          TEXT        NOT NULL,
    model         TEXT        NOT NULL,
    mileage_km    INTEGER     NOT NULL DEFAULT 0,
    created_at    TIMESTAMPTZ NOT NULL DEFAULT now()
);

CREATE TABLE IF NOT EXISTS orders (
    id                  BIGINT GENERATED ALWAYS AS IDENTITY PRIMARY KEY,
    order_number        TEXT        NOT NULL UNIQUE,
    customer_id         BIGINT      NOT NULL REFERENCES customers (id),
    vehicle_id          BIGINT      NOT NULL REFERENCES vehicles (id),
    preferred_date      DATE,
    problem_description TEXT        NOT NULL DEFAULT '',
    status              TEXT        NOT NULL DEFAULT 'angefragt',
    created_at          TIMESTAMPTZ NOT NULL DEFAULT now(),
    updated_at          TIMESTAMPTZ NOT NULL DEFAULT now()
);

CREATE INDEX IF NOT EXISTS idx_orders_status ON orders (status);
CREATE INDEX IF NOT EXISTS idx_orders_vehicle ON orders (vehicle_id);

-- Positions captured by the workshop while working on an order.
CREATE TABLE IF NOT EXISTS order_items (
    id               BIGINT GENERATED ALWAYS AS IDENTITY PRIMARY KEY,
    order_id         BIGINT      NOT NULL REFERENCES orders (id) ON DELETE CASCADE,
    kind             TEXT        NOT NULL CHECK (kind IN ('labor', 'part')),
    description      TEXT        NOT NULL,
    quantity         INTEGER     NOT NULL DEFAULT 1,
    unit_price_cents INTEGER     NOT NULL DEFAULT 0,
    total_cents      INTEGER     NOT NULL DEFAULT 0,
    created_at       TIMESTAMPTZ NOT NULL DEFAULT now()
);

CREATE INDEX IF NOT EXISTS idx_order_items_order ON order_items (order_id);

CREATE TABLE IF NOT EXISTS employees (
    id            BIGINT GENERATED ALWAYS AS IDENTITY PRIMARY KEY,
    name          TEXT        NOT NULL,
    email         TEXT        NOT NULL UNIQUE,
    password_hash TEXT        NOT NULL,
    created_at    TIMESTAMPTZ NOT NULL DEFAULT now()
);

CREATE TABLE IF NOT EXISTS employee_sessions (
    id          BIGINT GENERATED ALWAYS AS IDENTITY PRIMARY KEY,
    employee_id BIGINT      NOT NULL REFERENCES employees (id) ON DELETE CASCADE,
    token_hash  TEXT        NOT NULL UNIQUE,
    expires_at  TIMESTAMPTZ NOT NULL,
    created_at  TIMESTAMPTZ NOT NULL DEFAULT now()
);

CREATE INDEX IF NOT EXISTS idx_employee_sessions_employee ON employee_sessions (employee_id);

-- One invoice per order (UNIQUE order_id): re-finishing an order must not
-- produce a second invoice.
CREATE TABLE IF NOT EXISTS invoices (
    id           BIGINT GENERATED ALWAYS AS IDENTITY PRIMARY KEY,
    order_id     BIGINT      NOT NULL UNIQUE REFERENCES orders (id) ON DELETE CASCADE,
    order_number TEXT        NOT NULL,
    net_cents    INTEGER     NOT NULL DEFAULT 0,
    vat_cents    INTEGER     NOT NULL DEFAULT 0,
    gross_cents  INTEGER     NOT NULL DEFAULT 0,
    created_at   TIMESTAMPTZ NOT NULL DEFAULT now()
);

CREATE TABLE IF NOT EXISTS invoice_items (
    id               BIGINT GENERATED ALWAYS AS IDENTITY PRIMARY KEY,
    invoice_id       BIGINT  NOT NULL REFERENCES invoices (id) ON DELETE CASCADE,
    kind             TEXT    NOT NULL CHECK (kind IN ('labor', 'part')),
    description      TEXT    NOT NULL,
    quantity         INTEGER NOT NULL DEFAULT 1,
    unit_price_cents INTEGER NOT NULL DEFAULT 0,
    total_cents      INTEGER NOT NULL DEFAULT 0
);

CREATE INDEX IF NOT EXISTS idx_invoice_items_invoice ON invoice_items (invoice_id);

-- Outbox: the worker writes a notification row instead of sending real email.
CREATE TABLE IF NOT EXISTS notifications (
    id         BIGINT GENERATED ALWAYS AS IDENTITY PRIMARY KEY,
    order_id   BIGINT      REFERENCES orders (id) ON DELETE SET NULL,
    channel    TEXT        NOT NULL DEFAULT 'email',
    recipient  TEXT,
    subject    TEXT,
    body       TEXT,
    status     TEXT        NOT NULL DEFAULT 'queued',
    created_at TIMESTAMPTZ NOT NULL DEFAULT now()
);

CREATE INDEX IF NOT EXISTS idx_notifications_order ON notifications (order_id);

CREATE TABLE IF NOT EXISTS status_history (
    id              BIGINT GENERATED ALWAYS AS IDENTITY PRIMARY KEY,
    order_id        BIGINT      NOT NULL REFERENCES orders (id) ON DELETE CASCADE,
    status          TEXT        NOT NULL,
    previous_status TEXT,
    changed_at      TIMESTAMPTZ NOT NULL DEFAULT now()
);

CREATE INDEX IF NOT EXISTS idx_status_history_order ON status_history (order_id, changed_at);
