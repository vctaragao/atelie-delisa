<script>
  import OrderModal from './OrderModal.svelte'
  import AvisarWhatsapp from '../lib/AvisarWhatsapp.svelte'
  import ReceberNaRetirada from '../lib/ReceberNaRetirada.svelte'
  import PagamentoModal from './PagamentoModal.svelte'
  import { api } from '../lib/api.js'
  import { data, mutate } from '../lib/data.svelte.js'
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
    atrasado,
    todasAsEntregas,
    retiradaDaPeca,
    precisaConferir,
    DIAS_DA_SEMANA,
    ORDER_STATUSES,
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

  // O pedido cuja peça acabou de sair com saldo aberto: a pergunta "a cliente
  // pagou?" fica aberta sobre a agenda até ela responder.
  let cobrando = $state(null)
  let pagando = $state(null)

  const emAberto = (o) => !CONCLUIDOS.includes(o.status)
  const visivel = (o) => mostrarConcluidos || emAberto(o)

  /* A agenda trabalha com entregas, não com pedidos.

     Uma entrega é um pedido num dia: as peças dele que vencem naquela data e
     hora. O pedido com tudo para o mesmo dia tem uma entrega só e aparece
     como antes; o que tem a calça para sexta e o vestido para segunda aparece
     nos dois dias, cada um mostrando a peça daquele dia. É o que faz as datas
     por peça servirem para planejar a semana.

     Cada entrega é uma cópia do pedido com a data e as peças daquele
     compromisso, então tudo o que a tela já desenhava continua valendo. O
     pedido de verdade vem em `entrega.pedido`, e é ele que vai para a tela de
     edição: salvar a cópia, que tem só parte dos serviços, apagaria o resto. */
  const entregas = $derived(todasAsEntregas(data.orders))

  // Todas as entregas com data, agrupadas pelo dia. O calendário lê este mapa
  // quadradinho por quadradinho, em vez de varrer a lista inteira 42 vezes.
  const porDia = $derived.by(() => {
    const mapa = new Map()
    for (const e of entregas) {
      if (!e.due || !visivel(e)) continue
      if (!mapa.has(e.due)) mapa.set(e.due, [])
      mapa.get(e.due).push(e)
    }
    for (const doDia of mapa.values()) doDia.sort(porHora)
    return mapa
  })

  const semanas = $derived(monthGrid(mes))
  const doDia = (iso) => porDia.get(iso) || []

  const atrasados = $derived(
    entregas.filter((e) => atrasado(e, hoje)).sort((a, b) =>
      a.due === b.due ? porHora(a, b) : a.due < b.due ? -1 : 1,
    ),
  )
  const semData = $derived(entregas.filter((e) => !e.due && visivel(e)))

  const deHoje = $derived(entregas.filter((e) => e.due === hoje && emAberto(e)))
  const doMes = $derived(
    [...porDia].filter(([dia]) => monthOf(dia) === mes).flatMap(([, pedidos]) => pedidos),
  )
  const selecionados = $derived(diaSelecionado ? doDia(diaSelecionado) : [])

  const pecas = (lista) => lista.reduce((sum, e) => sum + orderQty(e), 0)

  /** "3 entregas · 7 peças" — o resumo que vai nos títulos.

      Conta entregas, e não pedidos: o pedido que tem peças para dois dias
      aparece nos dois, e chamar isso de "2 pedidos" daria a entender que há
      dois pedidos quando há um. */
  function resumo(lista) {
    const e = lista.length === 1 ? '1 entrega' : `${lista.length} entregas`
    const n = pecas(lista)
    return `${e} · ${n === 1 ? '1 peça' : `${n} peças`}`
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

  // A etapa muda nas listas da agenda sem abrir o pedido: é aqui que ela
  // acompanha o dia, e marcar a peça como pronta na mesma tela evita o
  // passeio até a lista de ordens.
  async function mudarEtapa(order, caixinha) {
    const status = caixinha.value
    if (status === order.status) return
    // A caixinha já mostra a etapa nova antes do servidor responder. Se a
    // gravação falhar, o pedido continua na etapa antiga: a caixinha volta
    // para ela, para a tela não afirmar uma coisa que não foi gravada.
    if (!(await mutate(() => api.setOrderStatus(order.id, status)))) {
      caixinha.value = order.status
      return
    }
    // Mover o pedido para "Entregue" é dizer que ele saiu do ateliê inteiro,
    // e vale a mesma pergunta da caixinha da peça.
    if (status === 'Entregue') perguntarDoPagamento(order.id)
  }

  /* Baixa da retirada, peça por peça, sem abrir o pedido: é o que ela faz com
     a cliente na frente, levando a calça e deixando o vestido para depois.

     A data gravada é hoje, que é o dia em que a peça está saindo. Desmarcar
     apaga a data — a baixa feita por engano tem volta. Como na troca de
     etapa, a caixinha volta ao que estava se a gravação falhar, para a tela
     não afirmar que a peça saiu quando o banco não registrou. */
  async function darBaixa(entrega, peca, caixinha) {
    const levou = caixinha.checked
    const ok = await mutate(() =>
      api.setPieceDelivered(entrega.id, peca, levou ? hoje : ''),
    )
    if (!ok) {
      caixinha.checked = !levou
      return
    }
    if (levou) perguntarDoPagamento(entrega.id)
  }

  /* A pergunta do pagamento, no instante em que a peça sai.

     É aqui que a baixa se perdia: o dinheiro é recebido no balcão e o
     lançamento ficava para "depois", que às vezes não vinha — e dias depois a
     mesma cliente era cobrada de novo. Marcar a retirada é o único momento em
     que o sistema sabe que a peça está saindo, com a cliente ainda na frente
     dela. É o momento de perguntar.

     Só pergunta quando há o que receber. O pedido já quitado, que é a maioria,
     não interrompe nada. O que já foi combinado como "paga depois" também não
     pergunta de novo: ela já respondeu uma vez. */
  function perguntarDoPagamento(orderID) {
    const atual = data.orders.find((o) => o.id === orderID)
    if (atual && precisaConferir(atual)) cobrando = atual
  }
</script>

<!-- As peças de uma entrega, cada uma com a caixinha de "a cliente levou".
     As três listas da tela — atrasadas, do dia e sem data — mostram a mesma
     coisa, e a baixa precisa estar disponível nas três: a peça atrasada é
     justamente a que a cliente vem buscar. -->
{#snippet pecasDaEntrega(entrega)}
  {#each groupItemsByPiece(entrega.items) as g (g.peca)}
    {@const levou = retiradaDaPeca(g.itens)}
    <label class="peca-levou" class:levou={Boolean(levou)}>
      <input
        type="checkbox"
        checked={Boolean(levou)}
        onchange={(ev) => darBaixa(entrega, g.peca, ev.currentTarget)}
        aria-label="A cliente levou {g.peca || 'a peça'} do pedido {entrega.number}"
      />
      <span>
        {#if g.peca}<b class="item-peca">{g.peca}:</b>{/if}
        {listItems(g.itens)}
        {#if levou}
          <span class="agenda-resumo">· levou em {fmtDate(levou)}</span>
        {/if}
      </span>
    </label>
  {/each}
{/snippet}

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
          {#each atrasados as o (o.chave)}
            <tr>
              <td>
                {fmtDate(o.due)}
                {#if o.dueTime}<br /><span class="agenda-hora">{o.dueTime}</span>{/if}
              </td>
              <td>
                <b>#{o.number}</b>
                {#if o.parcial}
                  <span class="entrega-parcial" title="Este pedido tem peças em mais de um dia; aqui está a parte deste dia">parcial</span>
                {/if}
              </td>
              <td>{o.clientName}</td>
              <td style="white-space:normal;min-width:170px">
                {@render pecasDaEntrega(o)}
              </td>
              <td><b>{orderQty(o)}</b></td>
              <td>
                <select
                  class="etapa-select {statusClass(o.status)}"
                  value={o.status}
                  onchange={(e) => mudarEtapa(o, e.currentTarget)}
                  aria-label="Etapa do pedido {o.number}"
                >
                  {#each ORDER_STATUSES as s (s)}
                    <option value={s}>{s}</option>
                  {/each}
                </select>
              </td>
              <td>{money(o.totalCents)}</td>
              <td>
                <div class="actions">
                  <button class="btn light small" onclick={() => (editing = o.pedido)}>Editar</button>
                  <button class="btn light small" onclick={() => (avisando = o.pedido)}>Avisar</button>
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
          {#each pedidos.slice(0, POR_DIA_NO_QUADRO) as o (o.chave)}
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
          {#each selecionados as o (o.chave)}
            <tr>
              <td><b class="agenda-hora">{fmtTime(o.dueTime)}</b></td>
              <td>
                <b>#{o.number}</b>
                {#if o.parcial}
                  <span class="entrega-parcial" title="Este pedido tem peças em mais de um dia; aqui está a parte deste dia">parcial</span>
                {/if}
              </td>
              <td>{o.clientName}</td>
              <td style="white-space:normal;min-width:170px">
                {@render pecasDaEntrega(o)}
              </td>
              <td><b>{orderQty(o)}</b></td>
              <td>
                <select
                  class="etapa-select {statusClass(o.status)}"
                  value={o.status}
                  onchange={(e) => mudarEtapa(o, e.currentTarget)}
                  aria-label="Etapa do pedido {o.number}"
                >
                  {#each ORDER_STATUSES as s (s)}
                    <option value={s}>{s}</option>
                  {/each}
                </select>
              </td>
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
                  <button class="btn light small" onclick={() => (editing = o.pedido)}>Editar</button>
                  <button class="btn light small" onclick={() => (avisando = o.pedido)}>Avisar</button>
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
          {#each semData as o (o.chave)}
            <tr>
              <td>
                <b>#{o.number}</b>
                {#if o.parcial}
                  <span class="entrega-parcial" title="Este pedido tem peças em mais de um dia; aqui está a parte deste dia">parcial</span>
                {/if}
              </td>
              <td>{o.clientName}</td>
              <td style="white-space:normal;min-width:170px">
                {@render pecasDaEntrega(o)}
              </td>
              <td><b>{orderQty(o)}</b></td>
              <td>
                <select
                  class="etapa-select {statusClass(o.status)}"
                  value={o.status}
                  onchange={(e) => mudarEtapa(o, e.currentTarget)}
                  aria-label="Etapa do pedido {o.number}"
                >
                  {#each ORDER_STATUSES as s (s)}
                    <option value={s}>{s}</option>
                  {/each}
                </select>
              </td>
              <td>{money(o.totalCents)}</td>
              <td>
                <div class="actions">
                  <button class="btn light small" onclick={() => (editing = o.pedido)}>Editar</button>
                  <button class="btn light small" onclick={() => (avisando = o.pedido)}>Avisar</button>
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

<!-- A pergunta do pagamento, logo depois de a peça ser marcada como saída.
     "Pagou só uma parte" fecha esta e abre a tela de pagamentos, onde ela
     digita o valor recebido. -->
{#if cobrando}
  <ReceberNaRetirada
    order={cobrando}
    onclose={() => (cobrando = null)}
    onpagamentos={(pedido) => (pagando = pedido)}
  />
{/if}

{#if pagando}
  <PagamentoModal order={pagando} onclose={() => (pagando = null)} />
{/if}
