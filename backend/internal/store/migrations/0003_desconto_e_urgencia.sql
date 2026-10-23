-- Migração 0003 — desconto e acréscimo de urgência no pedido.
--
-- Os dois são percentuais inteiros aplicados sobre o subtotal dos serviços:
-- o desconto combinado com a cliente e o acréscimo das entregas em menos de
-- 24 h. Ficam em colunas do pedido, e não em itens inventados na lista, para
-- que o comprovante possa mostrar "subtotal, desconto, urgência, total".
--
-- A view order_totals é refeita aqui para já devolver o total líquido. Com
-- isso tudo que lê o total — lista de pedidos, agenda, dashboard, a receber
-- do financeiro — passa a contar desconto e urgência sem precisar repetir a
-- conta em cada consulta.

-- +goose Up
ALTER TABLE orders ADD COLUMN discount_percent INTEGER NOT NULL DEFAULT 0;
ALTER TABLE orders ADD COLUMN rush_percent INTEGER NOT NULL DEFAULT 0;

DROP VIEW IF EXISTS order_totals;

CREATE VIEW order_totals AS
WITH itens AS (
  SELECT o.id AS order_id,
         COALESCE(SUM(i.qty * i.price_cents), 0) AS items_cents,
         o.discount_percent AS discount_percent,
         o.rush_percent AS rush_percent
  FROM orders o
  LEFT JOIN order_items i ON i.order_id = o.id
  GROUP BY o.id
)
SELECT order_id,
       items_cents,
       -- O arredondamento é feito uma vez, sobre cada percentual, e o
       -- resultado volta a ser inteiro: o resto da conta continua em centavos
       -- cheios, sem fração de centavo circulando.
       CAST(ROUND(items_cents * discount_percent / 100.0) AS INTEGER) AS discount_cents,
       CAST(ROUND(items_cents * rush_percent / 100.0) AS INTEGER) AS rush_cents,
       items_cents
         - CAST(ROUND(items_cents * discount_percent / 100.0) AS INTEGER)
         + CAST(ROUND(items_cents * rush_percent / 100.0) AS INTEGER) AS total_cents
FROM itens;

-- +goose Down
DROP VIEW IF EXISTS order_totals;

CREATE VIEW order_totals AS
SELECT o.id AS order_id,
       COALESCE(SUM(i.qty * i.price_cents), 0) AS total_cents
FROM orders o
LEFT JOIN order_items i ON i.order_id = o.id
GROUP BY o.id;

ALTER TABLE orders DROP COLUMN rush_percent;
ALTER TABLE orders DROP COLUMN discount_percent;
