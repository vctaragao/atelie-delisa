<script>
  import OrderModal from './OrderModal.svelte'
  import { api } from '../lib/api.js'
  import { data, mutate } from '../lib/data.svelte.js'
  import { money, fmtDate, statusClass, ORDER_STATUSES } from '../lib/format.js'

  let query = $state('')
  let statusFilter = $state('')
  let editing = $state(null)
  let creating = $state(false)

  const filtered = $derived.by(() => {
    const q = query.trim().toLowerCase()
    return data.orders.filter((o) => {
      if (statusFilter && o.status !== statusFilter) return false
      if (!q) return true
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
  <select bind:value={statusFilter} style="width:170px">
    <option value="">Todos os status</option>
    {#each ORDER_STATUSES as s (s)}
      <option value={s}>{s}</option>
    {/each}
  </select>
  <button class="btn" onclick={() => (creating = true)} disabled={data.clients.length === 0}>
    + Novo pedido
  </button>
</div>

{#if data.clients.length === 0}
  <div class="notice">Cadastre um cliente antes de criar pedidos.</div>
{/if}

<div class="card">
  <div class="table-wrap">
    <table>
      <thead>
        <tr>
          <th>Pedido</th><th>Cliente</th><th>Peças/serviços</th>
          <th>Entrega</th><th>Status</th><th>Total</th><th></th>
        </tr>
      </thead>
      <tbody>
        {#each filtered as o (o.id)}
          <tr>
            <td><b>#{o.number}</b></td>
            <td>{o.clientName}</td>
            <td style="white-space:normal;min-width:180px">
              {#each o.items as item (item.id)}
                {item.qty}× {item.name}<br />
              {/each}
            </td>
            <td>{fmtDate(o.due)}</td>
            <td><span class="badge {statusClass(o.status)}">{o.status}</span></td>
            <td>
              {money(o.totalCents)}
              {#if o.paid}
                <span class="badge pago" style="margin-left:6px">pago</span>
              {/if}
            </td>
            <td>
              <div class="actions">
                <button class="btn light small" onclick={() => (editing = o)}>Editar</button>
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
</div>

{#if creating}
  <OrderModal order={null} onclose={() => (creating = false)} />
{/if}

{#if editing}
  <OrderModal order={editing} onclose={() => (editing = null)} />
{/if}
