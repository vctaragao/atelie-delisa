# Ateliê Delisa — instruções para o Claude Code

Sistema de gestão do ateliê (clientes, ordens de serviço, tabela de preços,
financeiro). Frontend Svelte 5 + Vite, backend Go (`net/http`), banco SQLite
embutido no processo do backend. Tudo roda em containers — ver `README.md`.

## Quem usa e como

A Maria toca o ateliê e **não é desenvolvedora**. Ela descreve o que precisa
em linguagem comum; escrever o código é seu trabalho. Isso muda algumas coisas:

- Explique em termos do ateliê — clientes, ordens, pagamentos —, não em termos
  de componentes, endpoints e tabelas. Cite arquivos só quando ela precisar
  abrir algum.
- Ela não vai lembrar de pedir commit, push ou PR. **Você oferece** (ver abaixo).
- Ela não tem como perceber que uma alteração vai quebrar a produção. Você tem.
  Quando o que ela pediu tiver esse risco, diga antes de fazer.
- Erro que aparecer na tela dela é problema seu para investigar, não para ela
  interpretar. Peça o print ou leia o log do container.
- **Como ela manda um print:** arrastando o arquivo de imagem para dentro da
  janela do Claude Code, que insere o caminho. `Ctrl+V` não funciona neste
  terminal — não insista nem peça para ela tentar. As capturas do Windows
  (`Win+PrtScn`) caem em `%USERPROFILE%\OneDrive\Imagens\Capturas de tela\`,
  porque o OneDrive assumiu a pasta de imagens. O caminho tem espaços e
  acentos, então arrastar evita erro de digitação.

## O ciclo de trabalho

1. Ela abre o atalho **atelie-delisa-dev** e o sistema sobe com hot reload.
2. Ela conversa com você e pede funcionalidades ou correções.
3. Você edita os arquivos na pasta de desenvolvimento, e ela vê o resultado
   na hora no navegador.
4. **Quando ela demonstrar que está satisfeita** — "ficou bom", "é isso",
   "pode subir", "agora sim" —, ofereça, numa frase só: rodar os testes,
   commitar, dar push e abrir o PR da `development` para a `main`.
5. Ela aprova o PR no GitHub e clica em **Create a merge commit**.
6. Ela abre o atalho **atelie-delisa-prod**, que puxa a `main` e reconstrói.

O passo 4 é seu. Sem ele, o trabalho fica só no disco dela e o GitHub não tem
nada para mergear — ela vai procurar o PR e não vai encontrar.

## As duas pastas

O repositório está em duas pastas ao mesmo tempo, via `git worktree`:

| Pasta | Branch | Serve para |
|---|---|---|
| `%USERPROFILE%\atelie-delisa` | `main` | **produção** — o sistema que o ateliê usa |
| `%USERPROFILE%\atelie-delisa-dev` | `development` | **desenvolvimento** — onde o trabalho acontece |

São a mesma história do git, com dois diretórios de trabalho. Por isso uma
branch só pode estar em checkout numa pasta por vez: `git checkout development`
na pasta de produção **falha**, e é assim mesmo.

## Regra de branch — a mais importante deste arquivo

**Nunca faça alterações na `main`. Toda alteração acontece na `development`.**

Antes de editar qualquer arquivo:

```bash
git branch --show-current
```

- Se responder `development` → pode trabalhar.
- Se responder `main` → **pare**. Você está na pasta de produção. Não mude de
  branch aqui: vá para a pasta de desenvolvimento, que já está na
  `development`:
  ```bash
  cd ~/atelie-delisa-dev
  ```
- Se for uma branch de feature criada a partir da `development` → pode trabalhar.

Regras que seguem dessa:

- **Nunca** rode `git commit` com a `main` em checkout.
- **Nunca** rode `git push origin main`. A `main` está protegida por um ruleset
  no GitHub: exige pull request e recusa force-push e deleção.
- Código só chega na `main` por **pull request** aprovado pela Maria.
- O único método de merge liberado é **merge commit**. Squash e rebase estão
  desabilitados de propósito: eles reescrevem os commits e desalinhariam a
  `development`, que é permanente, forçando um religamento manual a cada PR.
- Se a Maria pedir explicitamente algo na `main`, confirme antes de fazer:
  avise que isso contraria a regra deste arquivo e pergunte se é mesmo a intenção.

## Commits e PRs

- Não commite por conta própria, mas **ofereça** sempre que ela se disser
  satisfeita. A regra é não commitar escondido, não é deixá-la sem caminho.
- Antes de oferecer, rode os testes. Não ofereça commit com teste quebrado:
  diga o que quebrou.
- Mensagens em português, no imperativo: `adiciona filtro por data nas ordens`.
- No corpo, explique **por que**, não o que o diff já mostra.
- Não use `--no-verify` nem pule hooks.

## Alterações no banco: sempre uma migração

Esta é a regra que mais protege os dados do ateliê.

Qualquer mudança de estrutura — coluna nova, tabela nova, índice, alteração de
tipo — vai num arquivo novo em `backend/internal/store/migrations/`, no formato
do **goose**:

```sql
-- +goose Up
ALTER TABLE clients ADD COLUMN apelido TEXT NOT NULL DEFAULT '';

-- +goose Down
ALTER TABLE clients DROP COLUMN apelido;
```

O nome começa com a próxima versão: `0002_adiciona_apelido.sql`. O goose aplica
o que falta a cada start e registra em `goose_db_version`.

**Nunca edite uma migração já aplicada.** Ela não roda de novo, então a
alteração passaria a existir apenas em bancos criados do zero — ou seja,
funcionaria em desenvolvimento e não em produção. Para mudar algo, crie a
próxima migração.

O banco de desenvolvimento nasce vazio e o de produção tem os dados de verdade.
É essa diferença que torna o erro invisível para ela: em dev a tabela é criada
já com a coluna nova, e só em produção a falta aparece. Há um teste cobrindo
exatamente isso em `internal/store/migrate_test.go` — se você mexer no
mecanismo, ele precisa continuar passando.

Antes de uma alteração de banco chegar à produção, sugira que ela exporte um
backup pela tela de Backup.

## Testes

```powershell
docker compose -p atelie-delisa-dev exec backend go test ./...
```

Go e Node não estão instalados na máquina — os comandos rodam dentro dos
containers.

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

Se o build de produção falhar, os containers no ar continuam intactos — o
`up -d --build` aborta antes de trocar qualquer coisa. Um código quebrado não
derruba o ateliê.

**Para voltar atrás**, se algo ruim chegar à produção: reverta pelo GitHub
(botão *Revert* no PR, que abre outro PR) e rode o atalho de produção de novo.

## Estrutura

```
backend/
  cmd/server/                 entrypoint
  internal/api/               handlers, router, respostas HTTP
  internal/store/             acesso ao SQLite, models
  internal/store/migrations/  migrações do goose
frontend/
  src/routes/                 telas (Dashboard, Clientes, Ordens, ...)
  src/lib/                    api client, helpers de formatação, componentes
scripts/
  atelie-delisa-chat.bat      abre este chat na pasta de desenvolvimento
  boas-vindas.md              o que fazer ao abrir o chat, lido por ele
  atelie-delisa-dev.bat       sobe o ambiente de desenvolvimento
  atelie-delisa-prod.bat      atualiza a main e sobe a produção
  icones/                     os .ico dos atalhos, com os fontes e como
                              gerá-los de novo (ver icones/README.md)
```

## Detalhes que economizam tempo

- O hot reload funciona por **polling** (`poll = true` no `.air.toml`,
  `usePolling: true` no `vite.config.js`). Não remova: bind mount do Docker no
  Windows não emite eventos de arquivo, e sem polling nada recarrega.
- Mudança em `Dockerfile`, `docker-compose*.yml`, `go.mod` ou `package.json`
  **não** é pega pelo hot reload. Nesses casos é preciso reabrir o atalho.
- `.gitattributes` força **LF** em todos os arquivos, **menos** `.bat`, `.cmd`
  e `.ps1`, que precisam de CRLF para o `cmd.exe` não engasgar.
- Nos `.bat`, chame `curl` e `ping` por caminho absoluto em
  `%SystemRoot%\System32` e use `ping` para esperar — o `timeout.exe` aborta
  quando o stdin está redirecionado.
- Sem acentos dentro dos `.bat`: o `cmd.exe` erra ao interpretar caracteres
  multibyte.
