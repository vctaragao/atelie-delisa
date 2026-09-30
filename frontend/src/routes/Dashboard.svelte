<script>
  import { data } from '../lib/data.svelte.js'
  import { money, fmtDate, statusClass } from '../lib/format.js'

  let { goto } = $props()

  const s = $derived(data.summary)
</script>

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
    {#each s?.upcomingOrders ?? [] as o (o.id)}
      <div style="padding:12px 0;border-bottom:1px solid var(--border)">
        <b>{o.clientName}</b><br />
        <span style="font-size:12px;color:var(--muted)">
          #{o.number} · entrega {fmtDate(o.due)}
        </span>
      </div>
    {:else}
      <div class="empty">Nenhuma entrega próxima.</div>
    {/each}
  </div>
</div>
