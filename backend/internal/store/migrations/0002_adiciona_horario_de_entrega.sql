-- Migração 0002 — horário de entrega do pedido.
--
-- A data de entrega já existia em orders.due, mas a agenda do dia precisa
-- ordenar os pedidos pela hora combinada com a cliente, e não havia onde
-- guardá-la. Fica numa coluna separada, em formato HH:MM, e é opcional: os
-- pedidos antigos continuam válidos com o default vazio.

-- +goose Up
ALTER TABLE orders ADD COLUMN due_time TEXT NOT NULL DEFAULT '';

-- Entregas do dia saem ordenadas por data e hora; o índice de o.due sozinho
-- deixava a ordenação por hora para a memória.
CREATE INDEX IF NOT EXISTS idx_orders_due_time ON orders (due, due_time);

-- +goose Down
DROP INDEX IF EXISTS idx_orders_due_time;
ALTER TABLE orders DROP COLUMN due_time;
