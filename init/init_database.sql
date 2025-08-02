CREATE TABLE IF NOT EXISTS orders (
    order_uid text PRIMARY KEY,
    track_number TEXT NOT NULL,
    entry TEXT NOT NULL,
    date_created TIMESTAMP NOT NULL,
    delivery_service TEXT,
    shardkey TEXT,
    sm_id INTEGER,
    oof_shard TEXT
);

CREATE TABLE IF NOT EXISTS deliveries (
    id SERIAL PRIMARY KEY,
    order_uid text NOT NULL REFERENCES orders(order_uid) ON DELETE CASCADE,
    name TEXT,
    phone TEXT,
    zip TEXT,
    city TEXT,
    address TEXT,
    region TEXT,
    email TEXT
);

CREATE TABLE payments (
    transaction      TEXT PRIMARY KEY,
    request_id       TEXT,
    currency         VARCHAR(10),
    provider         TEXT,
    amount           INT,
    payment_dt       BIGINT,
    bank             TEXT,
    delivery_cost    INT,
    goods_total      INT,
    custom_fee       INT,
    order_uid        TEXT NOT NULL REFERENCES orders(order_uid) ON DELETE CASCADE
);

CREATE TABLE items (
    id SERIAL PRIMARY KEY,
    chrt_id BIGINT NOT NULL,
    track_number TEXT NOT NULL,
    price INT NOT NULL,
    rid TEXT NOT NULL,
    name TEXT NOT NULL,
    sale INT NOT NULL,
    size TEXT NOT NULL,
    total_price INT NOT NULL,
    nm_id BIGINT NOT NULL,
    brand TEXT NOT NULL,
    status INT NOT NULL,
    order_uid TEXT NOT NULL REFERENCES orders(order_uid) ON DELETE CASCADE
);

CREATE TABLE IF NOT EXISTS order_metadata (
    order_uid text PRIMARY KEY REFERENCES orders(order_uid) ON DELETE CASCADE,
    locale TEXT,
    internal_signature TEXT,
    customer_id TEXT
);
