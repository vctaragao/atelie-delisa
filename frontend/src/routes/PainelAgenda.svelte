<script>
  import { data, refresh } from '../lib/data.svelte.js'
  import {
    fmtTime,
    isoDate,
    addDays,
    statusClass,
    orderQty,
    groupItemsByPiece,
    listItems,
    porHora,
  } from '../lib/format.js'

  // A cópia de teste e o sistema de verdade rodam no mesmo navegador, e o
  // quadro é uma janela sem barra de endereço: sem a faixa listrada e sem o
  // nome na barra de tarefas não haveria como saber qual dos dois está na
  // parede.
  const ambienteDeTeste = import.meta.env.DEV
  document.title = ambienteDeTeste ? 'TESTE — Agenda do ateliê' : 'Agenda — Ateliê Delisa'

  // A janela nova carrega a página do zero, com o estado vazio: os pedidos
  // precisam ser lidos aqui também.
  refresh()

  // O quadro fica numa segunda tela, sem ninguém tocando nele. Recarregar
  // sozinho é a única forma de o pedido que ela acabou de cadastrar na outra
  // janela aparecer aqui.
  const INTERVALO_DADOS = 60_000
  const INTERVALO_RELOGIO = 20_000

  const CONCLUIDOS = ['Entregue', 'Cancelado']
  const emAberto = (o) => !CONCLUIDOS.includes(o.status)

  let agora = $state(new Date())

  // Aqui a data de hoje não é lida uma vez só, como na tela da agenda: este
  // quadro passa a noite aberto e precisa virar o dia por conta própria.
  const hoje = $derived(isoDate(agora))
  const amanha = $derived(addDays(hoje, 1))

  $effect(() => {
    const relogio = setInterval(() => (agora = new Date()), INTERVALO_RELOGIO)
    const dados = setInterval(() => refresh(), INTERVALO_DADOS)
    return () => {
      clearInterval(relogio)
      clearInterval(dados)
    }
  })

  // O dia de hoje mostra também o que já foi entregue, riscado: num quadro
  // de parede vale ver o que saiu, não só o que falta. O cancelado sai de
  // vista — não é mais compromisso de ninguém.
  const deHoje = $derived(
    data.orders.filter((o) => o.due === hoje && o.status !== 'Cancelado').sort(porHora),
  )
  const faltamHoje = $derived(deHoje.filter(emAberto))
  const deAmanha = $derived(
    data.orders.filter((o) => o.due === amanha && emAberto(o)).sort(porHora),
  )
  const atrasados = $derived(
    data.orders
      .filter((o) => o.due && o.due < hoje && emAberto(o))
      .sort((a, b) => (a.due === b.due ? porHora(a, b) : a.due < b.due ? -1 : 1)),
  )

  const pecas = (pedidos) => pedidos.reduce((sum, o) => sum + orderQty(o), 0)

  const maiuscula = (s) => s.charAt(0).toUpperCase() + s.slice(1)

  const dataLonga = $derived(
    maiuscula(agora.toLocaleDateString('pt-BR', { weekday: 'long', day: 'numeric', month: 'long' })),
  )
  const relogio = $derived(
    agora.toLocaleTimeString('pt-BR', { hour: '2-digit', minute: '2-digit' }),
  )

  /** "Calça jeans: 1× Bainha · Vestido: 2× Ajuste" — as peças do pedido numa
      linha só, que é o que cabe na largura de uma linha do quadro. */
  function pecasEServicos(o) {
    return groupItemsByPiece(o.items)
      .map((g) => (g.peca ? `${g.peca}: ${listItems(g.itens)}` : listItems(g.itens)))
      .join(' · ')
  }

  /** A data curta que aparece nas atrasadas, onde o dia importa. */
  const diaCurto = (iso) =>
    new Date(`${iso}T00:00:00`).toLocaleDateString('pt-BR', { day: '2-digit', month: '2-digit' })
</script>

{#if ambienteDeTeste}
  <div class="faixa-teste" aria-hidden="true"></div>
{/if}

<div class="painel">
  <header class="painel-topo">
    <div>
      <small>Agenda do ateliê</small>
      <h1>{dataLonga}</h1>
    </div>
    <div class="painel-relogio">{relogio}</div>
  </header>

  {#if data.error}
    <p class="painel-erro">
      Não foi possível ler os pedidos agora. O quadro tenta de novo em um minuto.
    </p>
  {/if}

  {#if data.loading}
    <p class="painel-vazio">Carregando…</p>
  {:else}
    <div class="painel-numeros">
      <div>
        <span>Faltam entregar hoje</span>
        <strong>{faltamHoje.length}</strong>
      </div>
      <div>
        <span>Peças de hoje</span>
        <strong>{pecas(faltamHoje)}</strong>
      </div>
      <div class:alerta={atrasados.length > 0}>
        <span>Atrasadas</span>
        <strong>{atrasados.length}</strong>
      </div>
      <div>
        <span>Amanhã</span>
        <strong>{deAmanha.length}</strong>
      </div>
    </div>

    {#if atrasados.length}
      <section class="painel-bloco atrasado">
        <h2>Atrasadas</h2>
        {#each atrasados as o (o.id)}
          <div class="painel-linha">
            <div class="painel-hora">
              {diaCurto(o.due)}
              {#if o.dueTime}<small>{o.dueTime}</small>{/if}
            </div>
            <div class="painel-quem">
              <strong>{o.clientName}</strong>
              <span>{pecasEServicos(o)}</span>
            </div>
            <div class="painel-fim">
              <span class="badge {statusClass(o.status)}">{o.status}</span>
              <span class="painel-qtd">{orderQty(o)} pç</span>
              <span class="painel-num">#{o.number}</span>
            </div>
          </div>
        {/each}
      </section>
    {/if}

    <section class="painel-bloco">
      <h2>Hoje</h2>
      {#if deHoje.length}
        {#each deHoje as o (o.id)}
          <div class="painel-linha" class:feito={!emAberto(o)}>
            <div class="painel-hora">{fmtTime(o.dueTime)}</div>
            <div class="painel-quem">
              <strong>{o.clientName}</strong>
              <span>{pecasEServicos(o)}</span>
            </div>
            <div class="painel-fim">
              <span class="badge {statusClass(o.status)}">{o.status}</span>
              <span class="painel-qtd">{orderQty(o)} pç</span>
              <span class="painel-num">#{o.number}</span>
            </div>
          </div>
        {/each}
      {:else}
        <p class="painel-vazio">Nenhuma entrega marcada para hoje.</p>
      {/if}
    </section>

    {#if deAmanha.length}
      <section class="painel-bloco">
        <h2>Amanhã</h2>
        {#each deAmanha as o (o.id)}
          <div class="painel-linha">
            <div class="painel-hora">{fmtTime(o.dueTime)}</div>
            <div class="painel-quem">
              <strong>{o.clientName}</strong>
              <span>{pecasEServicos(o)}</span>
            </div>
            <div class="painel-fim">
              <span class="badge {statusClass(o.status)}">{o.status}</span>
              <span class="painel-qtd">{orderQty(o)} pç</span>
              <span class="painel-num">#{o.number}</span>
            </div>
          </div>
        {/each}
      </section>
    {/if}

    <footer class="painel-rodape">
      O quadro se atualiza sozinho. Última leitura às {relogio}.
    </footer>
  {/if}
</div>
