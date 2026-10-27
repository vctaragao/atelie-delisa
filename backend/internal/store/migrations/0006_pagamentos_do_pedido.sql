-- Migração 0006 — um registro para cada pagamento recebido no pedido.
--
-- Até aqui o pedido tinha só um sim/não em orders.paid. No balcão a conta
-- raramente é assim: a cliente deixa a peça e paga um sinal, ou leva a peça
-- e paga depois, ou paga em duas vezes. Sem um lugar para cada recebimento
-- não havia como dar baixa de parte do valor, nem saber quanto falta receber
-- de cada pedido.
--
-- Cada linha de order_payments é um dinheiro que entrou: a data, o valor, a
-- forma e uma observação opcional. A soma delas é o que o pedido tem de
-- pago, e o que falta é o total menos essa soma.
--
-- A coluna orders.paid CONTINUA EXISTINDO, e o backend continua mantendo-a
-- em dia a partir da soma dos pagamentos. Ela é o caminho de volta: se este
-- código for revertido pelo GitHub, o código antigo volta a ler orders.paid
-- e encontra o valor certo. Apagar a coluna aqui deixaria a produção sem
-- volta, porque o goose só aplica migração para a frente — reverter o código
-- não desfaz o banco.

-- +goose Up
CREATE TABLE IF NOT EXISTS order_payments (
  id          INTEGER PRIMARY KEY AUTOINCREMENT,
  order_id    INTEGER NOT NULL REFERENCES orders(id) ON DELETE CASCADE,
  date        TEXT NOT NULL,
  value_cents INTEGER NOT NULL DEFAULT 0,
  method      TEXT NOT NULL DEFAULT '',
  notes       TEXT NOT NULL DEFAULT '',
  created_at  TEXT NOT NULL DEFAULT (datetime('now'))
);

CREATE INDEX IF NOT EXISTS idx_order_payments_order ON order_payments (order_id);

-- Os pedidos que já estavam marcados como pagos viram um recebimento do
-- valor cheio, na data de entrada do pedido — a única data que o banco tem
-- para eles. Sem isto, no dia em que esta migração subisse, o faturamento do
-- ateliê apareceria zerado e todo pedido pago passaria a mostrar "falta
-- receber".
INSERT INTO order_payments (order_id, date, value_cents, method, notes)
SELECT o.id,
       CASE WHEN o.date <> '' THEN o.date ELSE DATE('now') END,
       t.total_cents,
       o.payment,
       'Baixa que já constava no pedido antes do controle de pagamentos'
FROM orders o
JOIN order_totals t ON t.order_id = o.id
WHERE o.paid = 1 AND t.total_cents > 0;

-- +goose Down
DROP INDEX IF EXISTS idx_order_payments_order;
DROP TABLE IF EXISTS order_payments;
