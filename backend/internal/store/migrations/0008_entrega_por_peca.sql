-- Migração 0008 — data de entrega e retirada por peça.
--
-- Até aqui a entrega era do pedido inteiro: uma data, uma hora, e a etapa
-- "Entregue" valendo para tudo de uma vez. Mas o ateliê trabalha peça por
-- peça: a calça fica para sexta e o vestido para segunda, e a cliente passa
-- na sexta, leva a calça e volta depois pelo resto.
--
-- Então cada item do pedido passa a ter:
--
--   due, due_time   a entrega combinada para aquela peça. Vazio significa
--                   "segue o pedido": a data do pedido continua sendo a
--                   combinação geral, e a da peça é a exceção. É isso que
--                   faz esta migração não mexer em nada do que já existe —
--                   todos os itens nascem vazios e continuam valendo pela
--                   data do pedido, como antes.
--
--   delivered_at    o dia em que a cliente levou aquela peça. Vazio é "ainda
--                   está aqui". Guarda a data, e não um sim/não, porque
--                   depois ela precisa saber quando a peça saiu, não só que
--                   saiu.
--
-- Os três são do item, e não de uma tabela de peças, porque "peça" não é um
-- cadastro: é o texto que ela escreve no bloco do pedido ("calça jeans"), e
-- os serviços daquele bloco carregam esse nome. A tela preenche os três
-- campos iguais para todos os serviços da mesma peça, do mesmo jeito que já
-- faz com o nome dela.

-- +goose Up
ALTER TABLE order_items ADD COLUMN due TEXT NOT NULL DEFAULT '';
ALTER TABLE order_items ADD COLUMN due_time TEXT NOT NULL DEFAULT '';
ALTER TABLE order_items ADD COLUMN delivered_at TEXT NOT NULL DEFAULT '';

-- +goose Down
ALTER TABLE order_items DROP COLUMN delivered_at;
ALTER TABLE order_items DROP COLUMN due_time;
ALTER TABLE order_items DROP COLUMN due;
