<script>
  import { data } from '../lib/data.svelte.js'
  import {
    money,
    fmtDate,
    fmtDateTime,
    statusClass,
    todasAsEntregas,
    todasRetiradas,
    groupItemsByPiece,
    porHora,
    today,
    precisaConferir,
    faltaReceber,
  } from '../lib/format.js'

  let { goto } = $props()

  const s = $derived(data.summary)

  const CONCLUIDOS = ['Entregue', 'Cancelado']

  /* As próximas entregas, contadas pela entrega de cada peça — a mesma regra
     da agenda e do quadro da parede.

     Antes esta lista vinha pronta do servidor, ordenada pela data do pedido.
     Com a data por peça isso passou a mentir: a calça que sai amanhã ficaria
     escondida atrás da data do pedido, que é semana que vem. Quem sabe
     desdobrar o pedido em entregas é o ajudante que a agenda já usa, então a
     lista é montada aqui, dos mesmos pedidos que a tela já tem.

     Fica de fora o que já saiu do ateliê: etapa concluída, ou todas as peças
     daquela entrega retiradas. */
  const PROXIMAS = 5

  const proximasEntregas = $derived.by(() => {
    const hoje = today()
    return todasAsEntregas(data.orders)
      .filter(
        (e) =>
          e.due &&
          e.due >= hoje &&
          !CONCLUIDOS.includes(e.status) &&
          !todasRetiradas(e.items),
      )
      .sort((a, b) => (a.due === b.due ? porHora(a, b) : a.due < b.due ? -1 : 1))
      .slice(0, PROXIMAS)
  })

  /* Peças que saíram do ateliê sem pagamento lançado.

     O aviso fica no Dashboard porque é a primeira tela que ela abre. A lista
     em si mora no Financeiro, mas uma lista que depende de ela se lembrar de
     ir olhar não resolve o problema de quem esquece de dar baixa. */
  const aConferir = $derived(data.orders.filter(precisaConferir))

  const totalAConferir = $derived(
    aConferir.reduce((soma, o) => soma + faltaReceber(o), 0),
  )

  /** "Calça jeans, Vestido" — as peças que saem naquela entrega. */
  const pecasDaEntrega = (entrega) =>
    groupItemsByPiece(entrega.items)
      .map((g) => g.peca)
      .filter(Boolean)
      .join(', ')
</script>

{#if aConferir.length}
  <div class="notice urgencia conferir-aviso">
    <span>
      <b>
        {aConferir.length === 1
          ? '1 pedido saiu do ateliê'
          : `${aConferir.length} pedidos saíram do ateliê`}
      </b>
      sem pagamento lançado, somando {money(totalAConferir)}. Pode ser dívida —
      ou baixa que ficou para trás. Conferir antes de cobrar.
    </span>
    <button class="btn small" onclick={() => goto('financeiro')}>
      Conferir agora
    </button>
  </div>
{/if}

<div class="cards">
  <div class="card">
    <div class="metric-label">Faturamento recebido</div>
    <div class="metric">{money(s?.revenueCents)}</div>
  </div>
  <div class="card">
    <div class="metric-label">A receber</div>
    <div class="metric gold">{money(s?.receivableCents)}</div>
  </div>
  <div class="card">
    <div class="metric-label">Pedidos em andamento</div>
    <div class="metric">{s?.inProgress ?? 0}</div>
  </div>
  <div class="card">
    <div class="metric-label">Peças prontas</div>
    <div class="metric">{s?.ready ?? 0}</div>
  </div>
</div>

<div class="dashboard-grid">
  <div class="card">
    <div class="section-title">
      <h2>Pedidos recentes</h2>
      <button class="btn light small" onclick={() => goto('ordens')}>Ver todos</button>
    </div>
    <div class="table-wrap">
      <table>
        <thead>
          <tr><th>Pedido</th><th>Cliente</th><th>Entrega</th><th>Status</th><th>Total</th></tr>
        </thead>
        <tbody>
          {#each s?.recentOrders ?? [] as o (o.id)}
            <tr>
              <td>#{o.number}</td>
              <td>{o.clientName}</td>
              <td>{fmtDate(o.due)}</td>
              <td><span class="badge {statusClass(o.status)}">{o.status}</span></td>
              <td>{money(o.totalCents)}</td>
            </tr>
          {:else}
            <tr><td colspan="5" class="empty">Nenhum pedido cadastrado.</td></tr>
          {/each}
        </tbody>
      </table>
    </div>
  </div>

  <div class="card">
    <div class="section-title"><h2>Próximas entregas</h2></div>
    {#each proximasEntregas as e (e.chave)}
      <div style="padding:12px 0;border-bottom:1px solid var(--border)">
        <b>{e.clientName}</b>
        {#if e.parcial}
          <span class="entrega-parcial">parcial</span>
        {/if}
        <br />
        <span style="font-size:12px;color:var(--muted)">
          #{e.number} · entrega {fmtDateTime(e.due, e.dueTime)}
          {#if pecasDaEntrega(e)}<br />{pecasDaEntrega(e)}{/if}
        </span>
      </div>
    {:else}
      <div class="empty">Nenhuma entrega próxima.</div>
    {/each}
  </div>
</div>
