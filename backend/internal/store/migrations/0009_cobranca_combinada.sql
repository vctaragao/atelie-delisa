-- Migração 0009 — a diferença entre "não pagou" e "esqueci de lançar".
--
-- O pagamento acontece no balcão e o lançamento acontece no computador. No
-- vão entre os dois a baixa se perde: a cliente paga em dinheiro, leva a
-- peça, e o pedido continua marcado como "a receber". Dias depois a mesma
-- cliente é cobrada de novo, porque a tela não tinha como saber se aquele
-- saldo aberto era dívida de verdade ou uma baixa esquecida.
--
-- pay_later_at é a resposta dela à pergunta que o sistema passa a fazer na
-- hora em que a peça sai: a data em que ficou combinado que a cliente paga
-- depois. Preenchida, o saldo aberto é dívida confirmada. Vazia, com a peça
-- já fora do ateliê, o pedido entra na lista de conferência do Financeiro —
-- é justamente o caso que pode ser uma baixa que ninguém lançou.
--
-- Vazia é o estado de todo pedido que já existe, e é o que deve ser: nenhuma
-- cobrança dos pedidos antigos foi conferida por ninguém. Os que já saíram
-- devendo vão aparecer na lista de conferência no primeiro dia, e é esse o
-- ponto — é o passivo que o ateliê nunca pôde enxergar.

-- +goose Up
ALTER TABLE orders ADD COLUMN pay_later_at TEXT NOT NULL DEFAULT '';

-- +goose Down
ALTER TABLE orders DROP COLUMN pay_later_at;
