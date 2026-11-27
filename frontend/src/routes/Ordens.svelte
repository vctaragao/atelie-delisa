<script>
  import OrderModal from './OrderModal.svelte'
  import PagamentoModal from './PagamentoModal.svelte'
  import Comprovante from './Comprovante.svelte'
  import AvisarWhatsapp from '../lib/AvisarWhatsapp.svelte'
  import { api } from '../lib/api.js'
  import { data, mutate, refresh } from '../lib/data.svelte.js'
  import {
    money,
    fmtDateTime,
    statusClass,
    groupItemsByPiece,
    listItems,
    faltaReceber,
    temSaldoAberto,
    retiradaDaPeca,
    entregaDaPeca,
    contagemDeRetiradas,
    fmtDate,
    ORDER_STATUSES,
  } from '../lib/format.js'

  let query = $state('')
  let statusFilter = $state('')
  let editing = $state(null)
  let creating = $state(false)
  let printing = $state(null)
  let avisando = $state(null)
  let pagando = $state(null)

  // Só a busca por texto, sem a etapa nem o filtro de pendentes. É a base das
  // contagens das etapas: procurando por "Ana", cada etapa mostra quantos
  // pedidos da Ana estão nela.
  const encontrados = $derived.by(() => {
    const q = query.trim().toLowerCase()
    if (!q) return data.orders
    return data.orders.filter((o) => {
      const haystack = [
        `#${o.number}`,
        o.clientName,
        ...(o.items || []).map((i) => i.name),
      ]
        .join(' ')
        .toLowerCase()
      return haystack.includes(q)
    })
  })

  /* Filtro de pagamento pendente, por cima do filtro de etapa. É o que
     responde "quem está me devendo": o Financeiro mostra o total pendente, e
     aqui ela chega aos pedidos que formam esse total, com o botão de
     pagamentos à mão.

     Pedido cancelado fica fora: ninguém cobra o que não vai ser feito, e é a
     mesma regra que o temSaldoAberto já usa no resto do sistema. */
  let soPendentes = $state(false)

  const pendentes = $derived(encontrados.filter(temSaldoAberto))
  const totalPendente = $derived(
    pendentes.reduce((soma, o) => soma + faltaReceber(o), 0),
  )

  /* A lista sobre a qual as etapas são contadas. Com o filtro de pendentes
     ligado, as contagens contam só os pendentes: senão a etapa diria "12
     Pronto" e a tabela embaixo mostraria 3, e o número viraria um enigma. */
  const base = $derived(soPendentes ? pendentes : encontrados)

  const filtered = $derived(
    statusFilter ? base.filter((o) => o.status === statusFilter) : base,
  )

  // As etapas do pedido como botões, com quantos pedidos há em cada uma. É o
  // que mostra, de uma olhada, que o ateliê tem orçamentos esperando resposta
  // e peças prontas esperando a cliente buscar.
  const etapas = $derived([
    { id: '', label: 'Todos', conta: base.length },
    ...ORDER_STATUSES.map((s) => ({
      id: s,
      label: s,
      conta: base.filter((o) => o.status === s).length,
    })),
  ])

  // Trocar a etapa no <select> da linha mexe no DOM antes do servidor
  // responder. Quando a gravação falha, o pedido continua na etapa antiga mas
  // a caixinha já mostra a nova; subir este número recria a tabela e devolve
  // cada pedido ao que o servidor diz.
  let tabelaVersao = $state(0)

  async function mudarEtapa(order, status) {
    if (status === order.status) return
    const ok = await mutate(() => api.setOrderStatus(order.id, status))
    if (!ok) {
      await refresh()
      tabelaVersao++
    }
  }

  async function remove(order) {
    if (!confirm(`Excluir o pedido #${order.number}?`)) return
    await mutate(() => api.deleteOrder(order.id))
  }
</script>

<div class="toolbar">
  <input
    class="search"
    placeholder="Buscar por cliente, nº do pedido ou peça..."
    bind:value={query}
  />
  <!-- Sem cliente nenhum o pedido continua podendo começar: o próprio modal
       cadastra a cliente. -->
  <button class="btn" onclick={() => (creating = true)}>+ Novo pedido</button>
</div>

<div class="etapas" role="group" aria-label="Filtrar por etapa do pedido">
  {#each etapas as e (e.id)}
    <button
      class="etapa {e.id ? statusClass(e.id) : ''}"
      class:ativa={statusFilter === e.id}
      onclick={() => (statusFilter = e.id)}
    >
      {e.label}
      <span class="etapa-conta">{e.conta}</span>
    </button>
  {/each}

  <!-- O pagamento pendente não é uma etapa do pedido: é outro corte da mesma
       lista, e por isso fica separado, à direita, e some junto com a etapa. -->
  <button
    class="etapa pendente-filtro"
    class:ativa={soPendentes}
    onclick={() => (soPendentes = !soPendentes)}
    title="Mostrar só os pedidos com valor a receber"
  >
    💰 A receber
    <span class="etapa-conta">{pendentes.length}</span>
  </button>
</div>

{#if soPendentes}
  <div class="notice">
    Mostrando só os pedidos com pagamento pendente:
    <b>{pendentes.length}</b>
    {pendentes.length === 1 ? 'pedido' : 'pedidos'}, somando
    <b>{money(totalPendente)}</b> a receber. Pedido cancelado não entra na
    conta.
  </div>
{/if}

<div class="card">
  {#key tabelaVersao}
    <div class="table-wrap">
      <table>
        <thead>
          <tr>
            <th>Pedido</th><th>Cliente</th><th>Peças/serviços</th>
            <th>Entrega</th><th>Etapa</th><th>Valor e pagamento</th><th></th>
          </tr>
        </thead>
        <tbody>
          {#each filtered as o (o.id)}
            {@const falta = faltaReceber(o)}
            <tr>
              <td><b>#{o.number}</b></td>
              <td>{o.clientName}</td>
              <td style="white-space:normal;min-width:180px">
                {#each groupItemsByPiece(o.items) as g (g.peca)}
                  {@const levou = retiradaDaPeca(g.itens)}
                  {@const entrega = entregaDaPeca(o, g.itens)}
                  <div class="ordem-peca" class:levou={Boolean(levou)}>
                    {#if g.peca}<b class="item-peca">{g.peca}:</b>{/if}
                    {listItems(g.itens)}
                    <!-- A data só aparece quando é da peça; a do pedido já
                         está na coluna de entrega, ao lado. -->
                    {#if entrega.propria}
                      <span class="peca-data">{fmtDateTime(entrega.due, entrega.dueTime)}</span>
                    {/if}
                    {#if levou}
                      <span class="agenda-resumo">levou em {fmtDate(levou)}</span>
                    {/if}
                  </div>
                {/each}
              </td>
              <td>
                {fmtDateTime(o.due, o.dueTime)}
                <!-- Quantas peças já saíram. O pedido só está realmente
                     entregue quando a última sai, e a etapa ao lado não conta
                     isso: ela é do pedido inteiro. -->
                {#if contagemDeRetiradas(o).total > 1}
                  {@const c = contagemDeRetiradas(o)}
                  <br />
                  <span class="agenda-resumo">
                    {c.retiradas} de {c.total} peças entregues
                  </span>
                {/if}
              </td>
              <td>
                <!-- A etapa muda aqui mesmo, sem abrir o pedido: é o que ela
                     faz várias vezes por dia, peça por peça. -->
                <select
                  class="etapa-select {statusClass(o.status)}"
                  value={o.status}
                  onchange={(e) => mudarEtapa(o, e.currentTarget.value)}
                  aria-label="Etapa do pedido {o.number}"
                >
                  {#each ORDER_STATUSES as s (s)}
                    <option value={s}>{s}</option>
                  {/each}
                </select>
              </td>
              <td>
                {money(o.totalCents)}
                {#if o.discountPercent || o.rushPercent}
                  <span class="ajuste-total">
                    {#if o.discountPercent}
                      <span class="desconto">−{o.discountPercent}% desconto</span>
                    {/if}
                    {#if o.discountPercent && o.rushPercent}·{/if}
                    {#if o.rushPercent}
                      <span class="acrescimo">+{o.rushPercent}% urgência</span>
                    {/if}
                  </span>
                {/if}
                <span class="situacao-pagamento">
                  {#if o.status === 'Cancelado'}
                    <span class="agenda-resumo">pedido cancelado</span>
                  {:else if falta === 0}
                    <span class="badge pago">pago</span>
                  {:else if o.paidCents > 0}
                    <span class="devendo">falta {money(falta)}</span>
                    <span class="agenda-resumo">· {money(o.paidCents)} já pago</span>
                  {:else}
                    <span class="devendo">a receber</span>
                  {/if}
                </span>
              </td>
              <td>
                <div class="actions">
                  <button class="btn light small" onclick={() => (editing = o)}>Editar</button>
                  <button class="btn light small" onclick={() => (pagando = o)}>
                    Pagamentos
                  </button>
                  <button class="btn light small" onclick={() => (printing = o)}>
                    Comprovante
                  </button>
                  <button class="btn light small" onclick={() => (avisando = o)}>
                    Avisar
                  </button>
                  <button class="btn danger small" onclick={() => remove(o)}>Excluir</button>
                </div>
              </td>
            </tr>
          {:else}
            <tr><td colspan="7" class="empty">Nenhum pedido encontrado.</td></tr>
          {/each}
        </tbody>
      </table>
    </div>
  {/key}
</div>

{#if creating}
  <OrderModal order={null} onclose={() => (creating = false)} />
{/if}

{#if editing}
  <OrderModal order={editing} onclose={() => (editing = null)} />
{/if}

{#if pagando}
  <PagamentoModal order={pagando} onclose={() => (pagando = null)} />
{/if}

{#if printing}
  <Comprovante order={printing} onclose={() => (printing = null)} />
{/if}

{#if avisando}
  <AvisarWhatsapp order={avisando} onclose={() => (avisando = null)} />
{/if}
