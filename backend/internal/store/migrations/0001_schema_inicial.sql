-- Migração 0001 — estado inicial do banco.
--
-- Era a antiga schema.sql, aplicada inteira a cada start. Virou a primeira
-- migração do goose: como tudo aqui usa IF NOT EXISTS, aplicá-la a um banco
-- que já tinha essas tabelas é inofensivo.

-- +goose Up
CREATE TABLE IF NOT EXISTS clients (
  id         INTEGER PRIMARY KEY AUTOINCREMENT,
  name       TEXT NOT NULL,
  phone      TEXT NOT NULL DEFAULT '',
  address    TEXT NOT NULL DEFAULT '',
  notes      TEXT NOT NULL DEFAULT '',
  created_at TEXT NOT NULL DEFAULT (datetime('now'))
);

CREATE TABLE IF NOT EXISTS services (
  id          INTEGER PRIMARY KEY AUTOINCREMENT,
  name        TEXT NOT NULL,
  category    TEXT NOT NULL DEFAULT '',
  price_cents INTEGER NOT NULL DEFAULT 0,
  time        TEXT NOT NULL DEFAULT '',
  created_at  TEXT NOT NULL DEFAULT (datetime('now'))
);

CREATE TABLE IF NOT EXISTS orders (
  id         INTEGER PRIMARY KEY AUTOINCREMENT,
  number     TEXT NOT NULL UNIQUE,
  client_id  INTEGER NOT NULL REFERENCES clients(id),
  date       TEXT NOT NULL DEFAULT '',
  due        TEXT NOT NULL DEFAULT '',
  status     TEXT NOT NULL DEFAULT 'Orçamento',
  payment    TEXT NOT NULL DEFAULT 'Não informado',
  paid       INTEGER NOT NULL DEFAULT 0,
  notes      TEXT NOT NULL DEFAULT '',
  created_at TEXT NOT NULL DEFAULT (datetime('now'))
);

CREATE TABLE IF NOT EXISTS order_items (
  id          INTEGER PRIMARY KEY AUTOINCREMENT,
  order_id    INTEGER NOT NULL REFERENCES orders(id) ON DELETE CASCADE,
  service_id  INTEGER REFERENCES services(id) ON DELETE SET NULL,
  name        TEXT NOT NULL,
  qty         INTEGER NOT NULL DEFAULT 1,
  price_cents INTEGER NOT NULL DEFAULT 0,
  position    INTEGER NOT NULL DEFAULT 0
);

CREATE TABLE IF NOT EXISTS transactions (
  id          INTEGER PRIMARY KEY AUTOINCREMENT,
  date        TEXT NOT NULL,
  type        TEXT NOT NULL CHECK (type IN ('entrada', 'saida')),
  description TEXT NOT NULL,
  value_cents INTEGER NOT NULL DEFAULT 0,
  payment     TEXT NOT NULL DEFAULT '',
  created_at  TEXT NOT NULL DEFAULT (datetime('now'))
);

CREATE INDEX IF NOT EXISTS idx_orders_client ON orders (client_id);
CREATE INDEX IF NOT EXISTS idx_orders_due ON orders (due);
CREATE INDEX IF NOT EXISTS idx_order_items_order ON order_items (order_id);

-- Total de cada pedido, derivado dos itens. Evita guardar um total
-- desatualizado numa coluna de orders.
CREATE VIEW IF NOT EXISTS order_totals AS
SELECT o.id AS order_id,
       COALESCE(SUM(i.qty * i.price_cents), 0) AS total_cents
FROM orders o
LEFT JOIN order_items i ON i.order_id = o.id
GROUP BY o.id;

-- +goose Down
DROP VIEW IF EXISTS order_totals;
DROP INDEX IF EXISTS idx_order_items_order;
DROP INDEX IF EXISTS idx_orders_due;
DROP INDEX IF EXISTS idx_orders_client;
DROP TABLE IF EXISTS transactions;
DROP TABLE IF EXISTS order_items;
DROP TABLE IF EXISTS orders;
DROP TABLE IF EXISTS services;
DROP TABLE IF EXISTS clients;
