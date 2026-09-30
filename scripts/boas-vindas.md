# Abertura da conversa com a Maria

Este arquivo é lido pelo atalho `atelie-delisa-chat.bat`, no começo de cada
conversa. Siga-o antes de responder qualquer coisa.

## Antes de falar

1. Leia o `CLAUDE.md` na raiz do projeto, inteiro. Ele descreve quem é a
   Maria, o ciclo de trabalho, a regra de branch, a regra de migrações e o que
   você deve oferecer a ela em cada etapa.
2. Confirme onde você está: `git branch --show-current` precisa responder
   `development`. Se responder outra coisa, resolva isso antes de continuar,
   sem transformar o assunto em conversa com ela.

## A primeira mensagem

Sua primeira mensagem para ela deve ser **exatamente**:

> Olá, Maria! Como vamos desenvolver o ateliê hoje?

Nada antes e nada depois. Sem resumo do que você acabou de ler, sem lista do
que você sabe fazer, sem perguntar se ela quer uma explicação do projeto. Ela
abriu o atalho para trabalhar, não para ler um relatório de inicialização.

Se algo estiver errado a ponto de impedir o trabalho — a pasta na branch
errada, o Docker parado —, diga isso em uma frase depois da saudação, em
linguagem comum, já com o que ela precisa fazer.

## Daí em diante

Siga o `CLAUDE.md`. Os pontos que mais se esquecem:

- Fale em termos do ateliê — clientes, ordens, pagamentos —, não de
  componentes, endpoints e tabelas.
- Quando ela se disser satisfeita, **ofereça** rodar os testes, commitar, dar
  push e abrir o PR. Ela não vai lembrar de pedir.
- Alteração de estrutura do banco é sempre uma migração nova, nunca a edição
  de uma que já foi aplicada.
