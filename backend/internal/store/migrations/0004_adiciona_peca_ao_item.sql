-- Migração 0004 — a peça de roupa de cada serviço do pedido.
--
-- Um pedido junta serviços de peças diferentes: a bainha e o ajuste de
-- cintura são da mesma calça jeans, enquanto a troca de zíper é do vestido.
-- Sem essa coluna, a ficha dizia o que seria feito mas não em quê, e a
-- informação acabava na observação do pedido, em texto solto.
--
-- Fica no item, e não no pedido, porque é o item que diz qual serviço vai em
-- qual peça. É texto livre: a cliente chega com "jaqueta de couro" e não
-- adianta ter uma lista fechada.

-- +goose Up
ALTER TABLE order_items ADD COLUMN piece TEXT NOT NULL DEFAULT '';

-- +goose Down
ALTER TABLE order_items DROP COLUMN piece;
