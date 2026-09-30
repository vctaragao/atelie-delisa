# Ateliê Delisa — Gestão

Sistema de gestão do ateliê: clientes, ordens de serviço, tabela de preços e
financeiro. Reescrito a partir do arquivo único `Atelie_Delisa_Gestao.html`.

| Parte | Tecnologia | Container |
|---|---|---|
| Frontend | Svelte 5 + Vite | `frontend` |
| Backend | Go (stdlib `net/http`) | `backend` |
| Banco | SQLite (`modernc.org/sqlite`, Go puro) | dentro do `backend`, em volume |

**Por que o SQLite não tem container próprio:** ele é uma biblioteca embutida
no processo, não um servidor de banco como o PostgreSQL. Não existe nada para
"subir". O arquivo `.db` vive num volume Docker (`atelie-data`), que o backend
abre — os dados sobrevivem a `docker compose down` e a reconstruções de imagem.

## Instalação numa máquina ou numa conta nova

Este roteiro parte do zero e termina com os dois ambientes no ar e os três
atalhos na área de trabalho. Serve tanto para outro computador quanto para
outra conta de usuário do mesmo Windows — os dois casos dão o mesmo trabalho,
porque o Docker Desktop guarda os dados por conta de usuário.

### 1. O que instalar

| Programa | Para que |
|---|---|
| [Docker Desktop](https://www.docker.com/products/docker-desktop/) | roda tudo: frontend, backend e banco |
| [Git para Windows](https://git-scm.com/download/win) | clonar o repositório e publicar alterações |
| [Claude Code](https://docs.claude.com/en/docs/claude-code/setup) | só para o atalho de chat; o sistema roda sem ele |

Go e Node **não** precisam ser instalados: rodam dentro dos containers.

Numa conta de usuário nova do Windows, duas coisas costumam passar batido:

- A conta precisa estar no grupo local **`docker-users`**, e só um
  administrador pode incluí-la. Sem isso o `docker` responde que não
  conseguiu falar com o daemon.
- O Docker Desktop precisa ser aberto **uma vez** nessa conta, para montar a
  máquina virtual dele. Ele mantém uma instalação separada por conta de
  usuário, e é por isso que os dados do ateliê não atravessam a troca de conta
  sozinhos. Numa primeira instalação isso é indiferente, porque não há dados
  ainda; mudando de máquina com o ateliê em uso, ver o passo 5.

O instalador nativo do Claude Code coloca o executável em
`%USERPROFILE%\.local\bin\claude.exe`, que é onde o atalho de chat procura.
Instalado de outro jeito, basta que `claude` esteja no `PATH`.

Não há caminho de pasta escrito dentro dos scripts: cada um descobre o
repositório a partir de onde o próprio arquivo está. Nada a ajustar por conta
de usuário.

### 2. Clonar e criar as duas pastas

O projeto trabalha com **duas pastas ao mesmo tempo**, via `git worktree`: a
`main` na pasta de produção e a `development` na de desenvolvimento. São o
mesmo repositório com dois diretórios de trabalho.

```powershell
cd $env:USERPROFILE
git clone https://github.com/vctaragao/atelie-delisa.git
cd atelie-delisa
git worktree add ..\atelie-delisa-dev development
```

O `clone` já deixa a `main` em checkout — essa pasta é a produção. O
`worktree add` cria a pasta vizinha com a `development`; como a branch existe
no `origin`, o git liga as duas sozinho.

| Pasta | Branch | Serve para |
|---|---|---|
| `%USERPROFILE%\atelie-delisa` | `main` | produção — o sistema que o ateliê usa |
| `%USERPROFILE%\atelie-delisa-dev` | `development` | desenvolvimento |

**Por que worktree e não dois clones:** dois clones seriam dois repositórios
independentes, cada um com seu histórico, e nada impediria que divergissem em
silêncio. Com worktree há um histórico só. A contrapartida é que a mesma
branch não pode estar em checkout nas duas pastas — `git checkout development`
na pasta de produção falha, e é assim mesmo.

Confira com `git worktree list`: devem aparecer as duas pastas.

### 3. Subir os dois ambientes na primeira vez

```powershell
# produção, na pasta da main
cd $env:USERPROFILE\atelie-delisa
docker compose -p atelie-delisa-prod -f docker-compose.prod.yml up -d --build

# desenvolvimento, na pasta da development
cd $env:USERPROFILE\atelie-delisa-dev
docker compose -p atelie-delisa-dev up --build
```

A primeira execução demora alguns minutos: baixa as imagens e compila as
dependências. Depois disso, produção responde em **http://localhost:8081** e
desenvolvimento em **http://localhost:5173**.

O banco é criado vazio e as migrações rodam sozinhas a cada start, então não
há nada a preparar à mão.

**O `-p` não é opcional.** Ele é o que separa os dois ambientes: sem ele os
dois usariam o mesmo nome de projeto, e subir um derrubaria os containers do
outro e misturaria os bancos.

### 4. Criar os atalhos na área de trabalho

```powershell
$ws = New-Object -ComObject WScript.Shell
$desktop = [Environment]::GetFolderPath('Desktop')

$atalhos = @(
  @{ nome = 'atelie-delisa-prod'; pasta = "$env:USERPROFILE\atelie-delisa" },
  @{ nome = 'atelie-delisa-dev';  pasta = "$env:USERPROFILE\atelie-delisa-dev" },
  @{ nome = 'atelie-delisa-chat'; pasta = "$env:USERPROFILE\atelie-delisa-dev" }
)

foreach ($a in $atalhos) {
  $lnk = $ws.CreateShortcut("$desktop\$($a.nome).lnk")
  $lnk.TargetPath = "$($a.pasta)\scripts\$($a.nome).bat"
  $lnk.WorkingDirectory = "$($a.pasta)\scripts"
  $lnk.IconLocation = "$($a.pasta)\scripts\icones\$($a.nome).ico,0"
  $lnk.Save()
}
```

Três detalhes que o script resolve e que dão errado quando feitos à mão:

- **Cada atalho aponta para a cópia do script que está na pasta em que ele vai
  operar**: o de produção na pasta da `main`, os de desenvolvimento e de chat
  na pasta da `development`. Isso não é detalhe estético. Os scripts descobrem
  a pasta do repositório a partir de onde o próprio arquivo está, então o
  atalho de produção apontado para a pasta de desenvolvimento tentaria subir
  produção a partir da `development` — o script percebe e recusa, mas o atalho
  para de funcionar.
- **A área de trabalho pode não estar em `%USERPROFILE%\Desktop`.** Quando o
  OneDrive assume as pastas do usuário, ela vira
  `%USERPROFILE%\OneDrive\Área de Trabalho`. O
  `[Environment]::GetFolderPath('Desktop')` devolve a certa nos dois casos.
- **O ícone fica gravado como caminho**, não copiado para dentro do atalho. É
  por isso que os `.ico` moram no repositório, em `scripts\icones\` — se forem
  movidos, os atalhos perdem o ícone. Ver `scripts/icones/README.md`.

Na máquina da Maria os três atalhos têm nomes antigos e diferentes destes
(`atelie-delisa`, `[dev]atelie-delisa` e `Claude CLI`). Numa instalação nova
vale usar os nomes acima, que são os que o `CLAUDE.md` cita.

### 5. Levar os dados do ateliê, se houver dados a levar

**Numa primeira instalação não há o que fazer aqui.** O banco nasce vazio, as
migrações criam as tabelas no primeiro start e o ateliê começa a cadastrar do
zero. Pule para o passo 6.

Este passo vale quando a produção já estava em uso em outro lugar. Nesse caso,
o volume **não acompanha a troca de conta de usuário**: o Docker Desktop mantém
uma instalação por conta, e a produção da conta nova nasce vazia mesmo havendo
dados na antiga. Eles vão pela tela de **Backup**:

1. Na conta antiga, abra a produção e exporte na tela de Backup. Sai um
   arquivo `.json` com clientes, serviços, pedidos e lançamentos.
2. Guarde o arquivo num lugar que as duas contas alcancem — o OneDrive, ou
   `C:\Users\Public`.
3. Na conta nova, abra a produção (não o ambiente de teste) e restaure esse
   arquivo pela mesma tela.

**A restauração substitui tudo o que estiver no banco**, ela não mistura. Numa
produção recém-criada é o que se quer, mas confira que você está na porta
**8081** antes de restaurar, e não na 5173.

Se ao abrir a produção da conta nova os clientes já estiverem lá, o Docker
compartilhou os dados e não há nada a fazer.

### 6. Credencial do GitHub

Clonar não pede nada, porque o repositório é público. Mas **publicar** pede —
é o que o ciclo de trabalho do `CLAUDE.md` faz no fim, com commit, push e pull
request.

Para isso a conta nova precisa de uma credencial com permissão de escrita em
`vctaragao/atelie-delisa`. O caminho mais curto é instalar o
[GitHub CLI](https://cli.github.com/) e rodar `gh auth login`, que resolve o
`git push` e é o que abre os pull requests.

Se quem for usar a conta nova tiver conta própria no GitHub, ela precisa ser
adicionada como colaboradora do repositório. Sem isso o push é recusado, mesmo
com o login feito.

### 7. Conferir que ficou tudo de pé

```powershell
git worktree list                      # as duas pastas, nas duas branches
docker compose -p atelie-delisa-prod ps
docker compose -p atelie-delisa-dev ps
docker compose -p atelie-delisa-dev exec backend go test ./...
```

E abrir os dois endereços no navegador: **http://localhost:8081** (produção,
sem faixa) e **http://localhost:5173** (desenvolvimento, com a faixa âmbar e o
selo "ambiente de teste" na barra lateral). A faixa é a única diferença visível
entre os dois, e é de propósito: as duas cópias rodam no mesmo navegador, e
ninguém repara na porta.

## Rodando o dia a dia

Há um atalho por ambiente, criados no passo 4. Pela linha de comando:

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

`atelie-delisa-prod_atelie-data` guarda os **dados reais do ateliê**. O de
desenvolvimento é descartável.

Para parar o desenvolvimento: `Ctrl+C` interrompe o log, mas os containers
seguem de pé; para derrubar, `docker compose -p atelie-delisa-dev down`. A
produção roda em segundo plano e volta sozinha quando o computador reinicia.

Se o build de produção falhar, os containers no ar continuam intactos: o
`up -d --build` aborta antes de trocar qualquer coisa. Código quebrado não
derruba o ateliê.

### Editando o código

Com o ambiente de desenvolvimento rodando, edite os arquivos normalmente no
Windows, na pasta `atelie-delisa-dev`:

- **Frontend** (`frontend/src/...`): o Vite recarrega o navegador na hora.
- **Backend** (`backend/...`): o [air](https://github.com/air-verse/air)
  recompila e reinicia o servidor; acompanhe no terminal.

Os dois usam *polling* para detectar mudanças, porque o compartilhamento de
disco do Docker no Windows não emite eventos de arquivo. Isso custa um pouco de
CPU e adiciona até ~0,5 s de atraso.

Mudança em `Dockerfile`, `docker-compose*.yml`, `go.mod` ou `package.json`
**não** é pega pelo hot reload: nesses casos é preciso reabrir o atalho.

## Estrutura

```
atelie-delisa/
├── docker-compose.yml           desenvolvimento (hot reload)
├── docker-compose.prod.yml      uso normal (nginx + binário)
├── CLAUDE.md                    instruções que o Claude Code lê
├── backend/
│   ├── cmd/server/main.go       configuração e encerramento do servidor
│   ├── internal/api/            rotas, handlers, validação, CORS
│   └── internal/store/          SQLite: consultas, models, backup
│       └── migrations/          migrações do goose, aplicadas no start
├── frontend/
│   ├── src/lib/                 api.js, format.js, estado, Modal
│   └── src/routes/              uma página por seção do menu
└── scripts/
    ├── atelie-delisa-prod.bat   atualiza a main e sobe a produção
    ├── atelie-delisa-dev.bat    sobe o ambiente de desenvolvimento
    ├── atelie-delisa-chat.bat   abre o Claude Code na pasta de dev
    ├── boas-vindas.md           o que o chat faz ao abrir
    └── icones/                  os .ico dos atalhos e como gerá-los
```

## API

Base: `/api`. Todos os valores monetários são **centavos inteiros**
(`priceCents`, `valueCents`, `totalCents`) — R$ 25,00 é `2500`.

| Método | Rota | O que faz |
|---|---|---|
| GET | `/health` | verificação de saúde |
| GET | `/summary` | métricas do dashboard e do financeiro |
| GET POST | `/clients` | listar, criar |
| PUT DELETE | `/clients/{id}` | atualizar, excluir |
| GET POST | `/services` | listar, criar |
| PUT DELETE | `/services/{id}` | atualizar, excluir |
| GET POST | `/orders` | listar, criar |
| GET PUT DELETE | `/orders/{id}` | detalhar, atualizar, excluir |
| GET POST | `/transactions` | listar, criar |
| DELETE | `/transactions/{id}` | excluir |
| GET | `/backup` | exportar tudo em JSON |
| POST | `/backup/restore` | substituir tudo por um JSON |
| DELETE | `/data` | apagar tudo e recriar a tabela de preços |

Exemplo:

```powershell
curl http://localhost:8080/api/summary
```

## Backup

A página **Backup** exporta um JSON com tudo e restaura a partir dele — é o
caminho recomendado, inclusive para migrar de computador.

Para copiar o arquivo `.db` diretamente:

```powershell
docker compose -p atelie-delisa-prod -f docker-compose.prod.yml cp backend:/data/atelie.db .\atelie-backup.db
```

## Diferenças em relação ao HTML original

O comportamento visível é o mesmo, com quatro mudanças deliberadas:

1. **Os dados saíram do navegador.** Antes ficavam no `localStorage`, ou seja,
   presos a um navegador de uma máquina e apagados junto com o histórico.
   Agora ficam no SQLite, e a mesma base pode ser aberta de outro dispositivo
   da rede.
2. **Dinheiro em centavos inteiros.** O original somava `float`, onde
   `0.1 + 0.2 ≠ 0.3`; os totais desviavam alguns centavos conforme o pedido
   crescia.
3. **A página "Ordens de serviço" voltou a funcionar.** No HTML original havia
   um parêntese fechado no lugar errado dentro de `renderOrders()`, que fazia
   `.sort()` ser chamado sobre o resultado booleano do filtro — a página
   quebrava com `TypeError` ao ser aberta.
4. **Pedidos cancelados não entram no faturamento.** O original somava um
   pedido cancelado marcado como pago no total recebido.

Também mudaram, em menor escala: os números de pedido usam `MAX` em vez de
`COUNT` (apagar um pedido não faz o próximo reutilizar o número), e excluir um
serviço da tabela de preços não afeta o histórico, porque cada item de pedido
guarda o nome e o preço que valeram na hora.

## Problemas comuns

**"O sistema abre mas diz que não consegue falar com o servidor."**
O backend ainda está compilando na primeira execução. Veja o terminal do
`docker compose up`: espere a linha `API ouvindo em :8080`.

**Mudei um arquivo e nada aconteceu.**
Confirme que o container está de pé com
`docker compose -p atelie-delisa-dev ps`. Se o backend parou por erro de
compilação, o air mostra o erro no terminal e continua rodando a versão
anterior.

**Quero começar do zero.**
`docker compose -p atelie-delisa-dev down -v` remove os containers **e o
volume** do ambiente de desenvolvimento — apaga o banco de dev, que é
descartável.

Nunca rode isso com `-p atelie-delisa-prod`: esse volume guarda os dados reais
do ateliê. E nunca rode sem `-p`, porque aí o alvo depende da pasta em que você
está. Exporte um backup antes, de qualquer jeito.
