-- Migração 0007 — o texto pronto para mandar o orçamento.
--
-- As três mensagens que nasceram com a tela de Mensagens cobriam pedido
-- pronto, lembrete de entrega e cobrança: todas depois de o trabalho estar
-- combinado. Faltava a primeira conversa, a que manda o valor para a cliente
-- aprovar — e é justamente a etapa em que todo pedido começa ("Orçamento").
--
-- Os textos padrão são inseridos pelo backend apenas quando a tabela está
-- vazia. Em produção ela não está, então sem esta migração a mensagem nova
-- apareceria só em bancos criados do zero, isto é, em desenvolvimento.
--
-- Por isso a condição do WHERE: num banco que ainda não tem mensagem nenhuma
-- esta migração não faz nada, e quem insere os padrões (os quatro, com este
-- texto no meio) continua sendo o backend. Num banco que já tem, entra só a
-- mensagem que falta — as que a Maria escreveu ou editou ficam intactas.
--
-- O corpo abaixo é uma cópia de como o texto estava neste momento. A fonte
-- viva dele é DefaultMessages(), em internal/store/messages.go, que é o que o
-- botão "restaurar padrão" reinsere; se o texto mudar lá, esta migração não
-- muda — ela já rodou.
--
-- position 0 coloca a mensagem na frente das outras na lista do aviso, para
-- acompanhar a ordem das etapas do pedido.

-- +goose Up
-- +goose StatementBegin
INSERT INTO message_templates (name, body, suggest_status, position)
SELECT
  'Orçamento para aprovação',
  'Oi, {cliente}! Aqui é do Ateliê Delisa.

Fiz o orçamento do seu pedido #{pedido}:

{orcamento}

Total: {total}.
Prazo de entrega: {entrega}.

Se estiver tudo certo, me confirme por aqui que eu já começo. Se quiser mudar alguma coisa, a gente ajeita.',
  'Orçamento',
  0
WHERE EXISTS (SELECT 1 FROM message_templates)
  AND NOT EXISTS (SELECT 1 FROM message_templates WHERE suggest_status = 'Orçamento')
-- +goose StatementEnd

-- +goose Down
-- +goose StatementBegin
DELETE FROM message_templates
WHERE suggest_status = 'Orçamento' AND name = 'Orçamento para aprovação'
-- +goose StatementEnd
