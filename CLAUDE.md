# Ateliê Delisa — instruções para o Claude Code

Sistema de gestão do ateliê (clientes, ordens de serviço, tabela de preços,
financeiro). Frontend Svelte 5 + Vite, backend Go (`net/http`), banco SQLite
embutido no processo do backend. Tudo roda em containers — ver `README.md`.

## As duas pastas

O repositório está em duas pastas ao mesmo tempo, via `git worktree`:

| Pasta | Branch | Serve para |
|---|---|---|
| `C:\Users\Maria\atelie-delisa` | `main` | **produção** — o sistema que o ateliê usa |
| `C:\Users\Maria\atelie-delisa-dev` | `development` | **desenvolvimento** — onde o trabalho acontece |

São a mesma história do git, com dois diretórios de trabalho. Por isso uma
branch só pode estar em checkout numa pasta por vez: `git checkout development`
na pasta de produção **falha**, e é assim mesmo.

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
- Se responder `main` → **pare**. Você está na pasta de produção. Não mude de
  branch aqui: vá para a pasta de desenvolvimento, que já está na
  `development`:
  ```bash
  cd C:\Users\Maria\atelie-delisa-dev
  ```
- Se for uma branch de feature criada a partir da `development` → pode trabalhar.

Regras que seguem dessa:

- **Nunca** rode `git commit` com a `main` em checkout.
- **Nunca** rode `git push origin main`.
- Branch de feature, quando fizer sentido, sai **da `development`**, nunca da
  `main`, e é criada na pasta de desenvolvimento:
  `git checkout -b feat/nome-da-coisa`
- Código só chega na `main` por **pull request** a partir da `development`,
  aberto e aprovado pela Maria — não por push direto, não por merge local.
- Se a Maria pedir explicitamente algo na `main`, confirme antes de fazer:
  avise que isso contraria a regra deste arquivo e pergunte se é mesmo a intenção.

## Commits

- Não commite sem a Maria pedir. Faça a alteração e mostre o que mudou.
- Mensagens em português, no imperativo: `adiciona filtro por data nas ordens`.
- Não use `--no-verify` nem pule hooks.

## Rodando o projeto

Há um atalho na área de trabalho para cada ambiente, apontando para os scripts
em `scripts/`. Pela linha de comando:

```powershell
# desenvolvimento, na pasta atelie-delisa-dev
docker compose -p atelie-delisa-dev up --build

# produção, na pasta atelie-delisa
docker compose -p atelie-delisa-prod -f docker-compose.prod.yml up -d --build
```

| Ambiente | Endereço | Volume de dados |
|---|---|---|
| Desenvolvimento | http://localhost:5173 (API em :8080) | `atelie-delisa-dev_atelie-data` |
| Produção | http://localhost:8081 | `atelie-delisa-prod_atelie-data` |

**O `-p` não é opcional.** Ele separa os dois ambientes: sem ele, os dois
compartilhariam nome de projeto, e subir um derrubaria os containers do outro
e misturaria os bancos.

`atelie-delisa-prod_atelie-data` guarda os **dados reais do ateliê**. Nunca
apague esse volume sem avisar. O de dev é descartável.

## Estrutura

```
backend/
  cmd/server/         entrypoint
  internal/api/       handlers, router, respostas HTTP
  internal/store/     acesso ao SQLite, models, schema.sql
frontend/
  src/routes/         telas (Dashboard, Clientes, Ordens, Financeiro, ...)
  src/lib/            api client, helpers de formatação, componentes
scripts/
  atelie-delisa-dev.bat    sobe o ambiente de desenvolvimento
  atelie-delisa-prod.bat   atualiza a main e sobe a produção
```

## Detalhes que economizam tempo

- `.gitattributes` força **LF** em todos os arquivos, **menos** `.bat`, `.cmd`
  e `.ps1`, que precisam de CRLF para o `cmd.exe` não engasgar. Não converta
  Dockerfiles, `nginx.conf` ou `.air.toml` para CRLF — eles vão para containers
  Linux e quebram.
- `backend/go.sum` está no `.gitignore` e é gerado dentro do container. Isso
  torna o build não reprodutível; se o Go for instalado na máquina, vale
  versionar o arquivo e remover a linha do `.gitignore`.
- Go e Node **não** estão instalados na máquina. Comandos como `go build` ou
  `npm install` só funcionam dentro dos containers
  (`docker compose -p atelie-delisa-dev exec backend go ...`).
