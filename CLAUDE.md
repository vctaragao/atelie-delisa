# Ateliê Delisa — instruções para o Claude Code

Sistema de gestão do ateliê (clientes, ordens de serviço, tabela de preços,
financeiro). Frontend Svelte 5 + Vite, backend Go (`net/http`), banco SQLite
embutido no processo do backend. Tudo roda em containers — ver `README.md`.

## Regra de branch — a mais importante deste arquivo

**Nunca faça alterações na `main`. Toda alteração acontece na `development`.**

A `main` é a branch estável, o retrato do que funciona. A `development` é onde
o trabalho acontece. Mudança que entra direto na `main` não passa por nenhuma
conferência e não tem como ser revisada antes de virar "a versão boa".

Na prática, antes de editar qualquer arquivo:

```bash
git branch --show-current
```

- Se responder `development` → pode trabalhar.
- Se responder `main` → **pare** e mude de branch antes de tocar em qualquer coisa:
  ```bash
  git checkout development
  ```
- Se for uma branch de feature criada a partir da `development` → pode trabalhar.

Regras que seguem dessa:

- **Nunca** rode `git commit` com a `main` em checkout.
- **Nunca** rode `git push origin main`.
- Branch de feature, quando fizer sentido, sai **da `development`**, nunca da `main`:
  `git checkout development && git checkout -b feat/nome-da-coisa`
- Código só chega na `main` por **pull request** a partir da `development`,
  aberto e aprovado pela Maria — não por push direto, não por merge local.
- Se a Maria pedir explicitamente algo na `main`, confirme antes de fazer:
  avise que isso contraria a regra deste arquivo e pergunte se é mesmo a intenção.

## Commits

- Não commite sem a Maria pedir. Faça a alteração e mostre o que mudou.
- Mensagens em português, no imperativo: `adiciona filtro por data nas ordens`.
- Não use `--no-verify` nem pule hooks.

## Rodando o projeto

```powershell
docker compose up --build
```

- Sistema: http://localhost:5173
- API: http://localhost:8080/api/health

O banco fica no volume Docker `atelie-data` e sobrevive a `docker compose down`.
Não apague esse volume sem avisar — são os dados reais do ateliê.

## Estrutura

```
backend/
  cmd/server/         entrypoint
  internal/api/       handlers, router, respostas HTTP
  internal/store/     acesso ao SQLite, models, schema.sql
frontend/
  src/routes/         telas (Dashboard, Clientes, Ordens, Financeiro, ...)
  src/lib/            api client, helpers de formatação, componentes
```

## Detalhes que economizam tempo

- `.gitattributes` força **LF** em todos os arquivos. Não converta para CRLF —
  Dockerfiles, `nginx.conf` e `.air.toml` são copiados para containers Linux
  e quebram com CRLF.
- `backend/go.sum` está no `.gitignore` e é gerado dentro do container. Isso
  torna o build não reprodutível; se o Go for instalado na máquina, vale
  versionar o arquivo e remover a linha do `.gitignore`.
- Go e Node **não** estão instalados na máquina. Comandos como `go build` ou
  `npm install` só funcionam dentro dos containers
  (`docker compose exec backend go ...`).
