-- Migração 0005 — as mensagens prontas do WhatsApp.
--
-- Os três textos nasceram escritos no código, e mudá-los exigia mexer no
-- sistema. Quem fala com as clientes é a Maria, e as palavras precisam ser
-- dela — então os textos passam a morar no banco, editáveis pela tela.
--
-- O corpo da mensagem aceita apelidos entre chaves ({cliente}, {servicos},
-- {total}...), trocados pelos dados do pedido na hora de montar a conversa.
--
-- suggest_status liga uma mensagem a uma situação do pedido: ao avisar um
-- pedido "Pronto", é a mensagem de retirada que já vem escolhida. Vazio
-- significa que a mensagem não é sugerida sozinha, só aparece na lista.
--
-- A tabela nasce vazia e os textos padrão são inseridos pelo backend quando
-- não há nenhuma mensagem cadastrada, do mesmo jeito que a tabela de preços.

-- +goose Up
CREATE TABLE IF NOT EXISTS message_templates (
  id             INTEGER PRIMARY KEY AUTOINCREMENT,
  name           TEXT NOT NULL,
  body           TEXT NOT NULL,
  suggest_status TEXT NOT NULL DEFAULT '',
  position       INTEGER NOT NULL DEFAULT 0,
  created_at     TEXT NOT NULL DEFAULT (datetime('now'))
);

-- +goose Down
DROP TABLE IF EXISTS message_templates;
