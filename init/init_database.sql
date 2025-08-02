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

CREATE TABLE IF NOT EXISTS payments (
  id SERIAL PRIMARY KEY,
  order_uid text NOT NULL REFERENCES orders(order_uid) ON DELETE CASCADE,
  transaction TEXT,
  request_id TEXT,
  currency TEXT,
  provider TEXT,
  amount INTEGER,
  payment_dt BIGINT,
  bank TEXT,
  delivery_cost INTEGER,
  goods_total INTEGER,
  custom_fee INTEGER
);

CREATE TABLE IF NOT EXISTS items (
   id SERIAL PRIMARY KEY,
   order_uid text NOT NULL REFERENCES orders(order_uid) ON DELETE CASCADE,
   chrt_id BIGINT,
   track_number TEXT,
   price INTEGER,
   rid TEXT,
   name TEXT,
   sale INTEGER,
   size TEXT,
   total_price INTEGER,
   nm_id BIGINT,
   brand TEXT,
   status INTEGER
);

CREATE TABLE IF NOT EXISTS order_metadata (
    order_uid text PRIMARY KEY REFERENCES orders(order_uid) ON DELETE CASCADE,
    locale TEXT,
    internal_signature TEXT,
    customer_id TEXT
);
