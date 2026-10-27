<script>
  import OrderModal from './OrderModal.svelte'
  import AvisarWhatsapp from '../lib/AvisarWhatsapp.svelte'
  import { data } from '../lib/data.svelte.js'
  import {
    money,
    fmtDate,
    fmtTime,
    fmtDayTitle,
    fmtMonthTitle,
    monthGrid,
    monthOf,
    shiftMonth,
    statusClass,
    orderQty,
    today,
    groupItemsByPiece,
    listItems,
    porHora,
    faltaReceber,
    DIAS_DA_SEMANA,
  } from '../lib/format.js'

  let { goto } = $props()

  // Pedido entregue ou cancelado não é mais compromisso: fica fora do
  // calendário por padrão, e aparece quando ela quiser conferir o que já saiu.
  const CONCLUIDOS = ['Entregue', 'Cancelado']

  // Quantos pedidos cabem dentro de um quadradinho do calendário antes de
  // virar "+2 mais". Acima disso a linha do calendário cresce demais e o mês
  // deixa de caber na tela sem rolar.
  const POR_DIA_NO_QUADRO = 3

  // A data de hoje é lida uma vez, na abertura da tela. Recalcular a cada
  // mudança deixaria a tela aberta de madrugada mostrando o dia errado, mas
  // isso só acontece se ela não tocar em nada até a virada do dia — e aí uma
  // troca de aba já resolve.
  const hoje = today()

  let mes = $state(monthOf(hoje))
  let diaSelecionado = $state(hoje)
  let mostrarConcluidos = $state(false)
  let editing = $state(null)
  let avisando = $state(null)

  const emAberto = (o) => !CONCLUIDOS.includes(o.status)
  const visivel = (o) => mostrarConcluidos || emAberto(o)

  // Todos os pedidos com data, agrupados pelo dia da entrega. O calendário lê
  // este mapa quadradinho por quadradinho, em vez de varrer a lista inteira
  // de pedidos 42 vezes.
  const porDia = $derived.by(() => {
    const mapa = new Map()
    for (const o of data.orders) {
      if (!o.due || !visivel(o)) continue
      if (!mapa.has(o.due)) mapa.set(o.due, [])
      mapa.get(o.due).push(o)
    }
    for (const pedidos of mapa.values()) pedidos.sort(porHora)
    return mapa
  })

  const semanas = $derived(monthGrid(mes))
  const doDia = (iso) => porDia.get(iso) || []

  const atrasados = $derived(
    data.orders.filter((o) => o.due && o.due < hoje && emAberto(o)).sort((a, b) =>
      a.due === b.due ? porHora(a, b) : a.due < b.due ? -1 : 1,
    ),
  )
  const semData = $derived(data.orders.filter((o) => !o.due && visivel(o)))

  const deHoje = $derived(data.orders.filter((o) => o.due === hoje && emAberto(o)))
  const doMes = $derived(
    [...porDia].filter(([dia]) => monthOf(dia) === mes).flatMap(([, pedidos]) => pedidos),
  )
  const selecionados = $derived(diaSelecionado ? doDia(diaSelecionado) : [])

  const pecas = (pedidos) => pedidos.reduce((sum, o) => sum + orderQty(o), 0)

  /** "3 pedidos · 7 peças" — o resumo que vai nos títulos. */
  function resumo(pedidos) {
    const p = pedidos.length === 1 ? '1 pedido' : `${pedidos.length} pedidos`
    const n = pecas(pedidos)
    return `${p} · ${n === 1 ? '1 peça' : `${n} peças`}`
  }

  /** Troca o mês mostrado. O dia aberto embaixo acompanha: hoje, se o mês for
      o corrente; senão a primeira entrega do mês, para a tela nunca ficar com
      um dia de outro mês selecionado. */
  function irPara(novoMes) {
    mes = novoMes
    if (monthOf(hoje) === mes) {
      diaSelecionado = hoje
      return
    }
    const dias = [...porDia.keys()].filter((d) => monthOf(d) === mes).sort()
    diaSelecionado = dias[0] ?? null
  }

  /** Abre a agenda numa janela só dela, para ficar aberta numa segunda tela.
      A janela recebe um nome: clicar de novo traz de volta a mesma, em vez de
      empilhar um quadro novo a cada clique. */
  function abrirPainel() {
    const janela = window.open(
      `${location.pathname}?painel=agenda`,
      'atelie-painel-agenda',
      'popup=yes,width=1150,height=900',
    )
    janela?.focus()
  }
</script>

<div class="toolbar">
  <span style="flex:1"></span>
  <button class="btn light" onclick={abrirPainel}>⧉ Abrir quadro em outra janela</button>
</div>

<div class="cards">
  <div class="card">
    <div class="metric-label">Entregas de hoje</div>
    <div class="metric">{deHoje.length}</div>
  </div>
  <div class="card">
    <div class="metric-label">Peças para hoje</div>
    <div class="metric">{pecas(deHoje)}</div>
  </div>
  <div class="card">
    <div class="metric-label">Entregas atrasadas</div>
    <div class="metric" style={atrasados.length ? 'color:var(--danger)' : ''}>
      {atrasados.length}
    </div>
  </div>
  <div class="card">
    <div class="metric-label">Peças no mês</div>
    <div class="metric gold">{pecas(doMes)}</div>
  </div>
</div>

{#if atrasados.length}
  <div class="card agenda-dia atrasado">
    <div class="section-title">
      <h2>Entregas atrasadas</h2>
      <span class="agenda-resumo">{resumo(atrasados)}</span>
    </div>
    <div class="table-wrap">
      <table>
        <thead>
          <tr>
            <th>Entrega</th><th>Pedido</th><th>Cliente</th>
            <th>Peças e serviços</th><th>Qtd.</th><th>Status</th><th>Total</th><th></th>
          </tr>
        </thead>
        <tbody>
          {#each atrasados as o (o.id)}
            <tr>
              <td>
                {fmtDate(o.due)}
                {#if o.dueTime}<br /><span class="agenda-hora">{o.dueTime}</span>{/if}
              </td>
              <td><b>#{o.number}</b></td>
              <td>{o.clientName}</td>
              <td style="white-space:normal;min-width:170px">
                {#each groupItemsByPiece(o.items) as g (g.peca)}
                  {#if g.peca}<b class="item-peca">{g.peca}:</b>{/if}
                  {listItems(g.itens)}<br />
                {/each}
              </td>
              <td><b>{orderQty(o)}</b></td>
              <td><span class="badge {statusClass(o.status)}">{o.status}</span></td>
              <td>{money(o.totalCents)}</td>
              <td>
                <div class="actions">
                  <button class="btn light small" onclick={() => (editing = o)}>Editar</button>
                  <button class="btn light small" onclick={() => (avisando = o)}>Avisar</button>
                </div>
              </td>
            </tr>
          {/each}
        </tbody>
      </table>
    </div>
  </div>
{/if}

<div class="card">
  <div class="cal-topo">
    <div class="cal-nav">
      <button
        class="btn light small"
        aria-label="Mês anterior"
        onclick={() => irPara(shiftMonth(mes, -1))}
      >
        ‹
      </button>
      <h2>{fmtMonthTitle(mes)}</h2>
      <button
        class="btn light small"
        aria-label="Mês seguinte"
        onclick={() => irPara(shiftMonth(mes, 1))}
      >
        ›
      </button>
      {#if mes !== monthOf(hoje)}
        <button class="btn light small" onclick={() => irPara(monthOf(hoje))}>Hoje</button>
      {/if}
    </div>
    <div class="cal-topo-direita">
      <span class="agenda-resumo">{resumo(doMes)} no mês</span>
      <select bind:value={mostrarConcluidos} style="width:230px">
        <option value={false}>Só pedidos em aberto</option>
        <option value={true}>Incluir entregues e cancelados</option>
      </select>
    </div>
  </div>

  <div class="calendario">
    {#each DIAS_DA_SEMANA as dia (dia)}
      <div class="cal-cabecalho">{dia}</div>
    {/each}

    {#each semanas as semana, s (s)}
      {#each semana as celula (celula.iso)}
        {@const pedidos = doDia(celula.iso)}
        <button
          type="button"
          class="cal-dia"
          class:fora={!celula.noMes}
          class:hoje={celula.iso === hoje}
          class:selecionado={celula.iso === diaSelecionado}
          class:vazio={pedidos.length === 0}
          onclick={() => (diaSelecionado = celula.iso)}
        >
          <span class="cal-numero">
            {celula.numero}
            {#if pedidos.length}
              <span class="cal-contagem">{pecas(pedidos)} pç</span>
            {/if}
          </span>
          {#each pedidos.slice(0, POR_DIA_NO_QUADRO) as o (o.id)}
            <span class="cal-pedido {statusClass(o.status)}" title="#{o.number} · {o.clientName}">
              {#if o.dueTime}<b>{o.dueTime}</b>{/if}
              {o.clientName}
            </span>
          {/each}
          {#if pedidos.length > POR_DIA_NO_QUADRO}
            <span class="cal-mais">+{pedidos.length - POR_DIA_NO_QUADRO} mais</span>
          {/if}
        </button>
      {/each}
    {/each}
  </div>
</div>

<div class="card agenda-dia" class:hoje={diaSelecionado === hoje}>
  <div class="section-title">
    <h2>{diaSelecionado ? fmtDayTitle(diaSelecionado, hoje) : 'Nenhum dia escolhido'}</h2>
    <span class="agenda-resumo">
      {#if selecionados.length}{resumo(selecionados)}{/if}
    </span>
  </div>

  {#if selecionados.length}
    <div class="table-wrap">
      <table>
        <thead>
          <tr>
            <th>Horário</th><th>Pedido</th><th>Cliente</th>
            <th>Peças e serviços</th><th>Qtd.</th><th>Status</th><th>Total</th><th></th>
          </tr>
        </thead>
        <tbody>
          {#each selecionados as o (o.id)}
            <tr>
              <td><b class="agenda-hora">{fmtTime(o.dueTime)}</b></td>
              <td><b>#{o.number}</b></td>
              <td>{o.clientName}</td>
              <td style="white-space:normal;min-width:170px">
                {#each groupItemsByPiece(o.items) as g (g.peca)}
                  {#if g.peca}<b class="item-peca">{g.peca}:</b>{/if}
                  {listItems(g.itens)}<br />
                {/each}
              </td>
              <td><b>{orderQty(o)}</b></td>
              <td><span class="badge {statusClass(o.status)}">{o.status}</span></td>
              <td>
                {money(o.totalCents)}
                <!-- Quanto receber na entrega. É neste dia que a cliente vem
                     buscar a peça, e é a última hora de cobrar o que falta. -->
                <span class="situacao-pagamento">
                  {#if faltaReceber(o) === 0}
                    <span class="badge pago">pago</span>
                  {:else if o.paidCents > 0}
                    <span class="devendo">falta {money(faltaReceber(o))}</span>
                  {:else}
                    <span class="devendo">a receber</span>
                  {/if}
                </span>
              </td>
              <td>
                <div class="actions">
                  <button class="btn light small" onclick={() => (editing = o)}>Editar</button>
                  <button class="btn light small" onclick={() => (avisando = o)}>Avisar</button>
                </div>
              </td>
            </tr>
          {/each}
        </tbody>
      </table>
    </div>
  {:else}
    <div class="empty">
      {diaSelecionado
        ? 'Nenhuma entrega marcada para este dia.'
        : 'Clique num dia do calendário para ver as entregas.'}
    </div>
  {/if}
</div>

{#if semData.length}
  <div class="card agenda-dia">
    <div class="section-title">
      <h2>Sem data de entrega</h2>
      <span class="agenda-resumo">{resumo(semData)}</span>
    </div>
    <p class="agenda-aviso">
      Estes pedidos não têm dia marcado, então não aparecem no calendário.
      Abra cada um para combinar a data e o horário.
    </p>
    <div class="table-wrap">
      <table>
        <thead>
          <tr>
            <th>Pedido</th><th>Cliente</th><th>Peças e serviços</th>
            <th>Qtd.</th><th>Status</th><th>Total</th><th></th>
          </tr>
        </thead>
        <tbody>
          {#each semData as o (o.id)}
            <tr>
              <td><b>#{o.number}</b></td>
              <td>{o.clientName}</td>
              <td style="white-space:normal;min-width:170px">
                {#each groupItemsByPiece(o.items) as g (g.peca)}
                  {#if g.peca}<b class="item-peca">{g.peca}:</b>{/if}
                  {listItems(g.itens)}<br />
                {/each}
              </td>
              <td><b>{orderQty(o)}</b></td>
              <td><span class="badge {statusClass(o.status)}">{o.status}</span></td>
              <td>{money(o.totalCents)}</td>
              <td>
                <div class="actions">
                  <button class="btn light small" onclick={() => (editing = o)}>Editar</button>
                  <button class="btn light small" onclick={() => (avisando = o)}>Avisar</button>
                </div>
              </td>
            </tr>
          {/each}
        </tbody>
      </table>
    </div>
  </div>
{/if}

<div class="toolbar" style="margin-top:4px">
  <span style="flex:1"></span>
  <button class="btn light" onclick={() => goto('ordens')}>Ver todas as ordens</button>
</div>

{#if editing}
  <OrderModal order={editing} onclose={() => (editing = null)} />
{/if}

{#if avisando}
  <AvisarWhatsapp order={avisando} onclose={() => (avisando = null)} />
{/if}
