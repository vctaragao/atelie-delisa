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

## O que você precisa instalar

Só o **[Docker Desktop](https://www.docker.com/products/docker-desktop/)**.

Go e Node rodam dentro dos containers — não precisa instalar nenhum dos dois
na máquina.

## Rodando em desenvolvimento

```powershell
cd C:\Users\Maria\atelie-delisa
docker compose up --build
```

A primeira execução demora alguns minutos (baixa as imagens e as
dependências). Depois:

- Sistema: **http://localhost:5173**
- API: http://localhost:8080/api/health

Para parar: `Ctrl+C`, ou `docker compose down` em outro terminal. Os dados
ficam no volume.

### Editando o código

Com o `docker compose up` rodando, edite os arquivos normalmente no Windows:

- **Frontend** (`frontend/src/...`): o Vite recarrega o navegador na hora.
- **Backend** (`backend/...`): o [air](https://github.com/air-verse/air)
  recompila e reinicia o servidor; acompanhe no terminal.

Os dois usam *polling* para detectar mudanças, porque o compartilhamento de
disco do Docker no Windows não emite eventos de arquivo. Isso custa um pouco
de CPU e adiciona até ~0,5 s de atraso.

## Rodando em modo de uso (produção)

```powershell
docker compose -f docker-compose.prod.yml up -d --build
```

Acesse **http://localhost:8081**. O frontend vira arquivos estáticos servidos
pelo nginx, e o backend um binário estático. Sem hot reload, bem mais leve.

Para parar: `docker compose -f docker-compose.prod.yml down`

> Os dois ambientes usam volumes separados por padrão (o Compose prefixa o
> nome do projeto). Para compartilhar os dados, rode ambos com o mesmo
> projeto: `docker compose -p atelie ...`.

## Estrutura

```
atelie-delisa/
├── docker-compose.yml          desenvolvimento (hot reload)
├── docker-compose.prod.yml     uso normal (nginx + binário)
├── backend/
│   ├── cmd/server/main.go      configuração e encerramento do servidor
│   ├── internal/api/           rotas, handlers, validação, CORS
│   └── internal/store/         SQLite: schema, consultas, backup
└── frontend/
    ├── src/lib/                api.js, format.js, estado, Modal
    └── src/routes/             uma página por seção do menu
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
docker compose cp backend:/data/atelie.db .\atelie-backup.db
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
Confirme que o container está de pé (`docker compose ps`). Se o backend
parou por erro de compilação, o air mostra o erro no terminal e continua
rodando a versão anterior.

**Quero começar do zero.**
`docker compose down -v` remove os containers **e os volumes** — apaga o
banco. Exporte um backup antes.
