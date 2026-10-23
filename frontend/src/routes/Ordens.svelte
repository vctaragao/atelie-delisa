<script>
  import OrderModal from './OrderModal.svelte'
  import Comprovante from './Comprovante.svelte'
  import AvisarWhatsapp from '../lib/AvisarWhatsapp.svelte'
  import { api } from '../lib/api.js'
  import { data, mutate } from '../lib/data.svelte.js'
  import {
    money,
    fmtDateTime,
    statusClass,
    groupItemsByPiece,
    listItems,
    ORDER_STATUSES,
  } from '../lib/format.js'

  let query = $state('')
  let statusFilter = $state('')
  let editing = $state(null)
  let creating = $state(false)
  let printing = $state(null)
  let avisando = $state(null)

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
  <!-- Sem cliente nenhum o pedido continua podendo começar: o próprio modal
       cadastra a cliente. -->
  <button class="btn" onclick={() => (creating = true)}>+ Novo pedido</button>
</div>

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
              {#each groupItemsByPiece(o.items) as g (g.peca)}
                {#if g.peca}<b class="item-peca">{g.peca}:</b>{/if}
                {listItems(g.itens)}<br />
              {/each}
            </td>
            <td>{fmtDateTime(o.due, o.dueTime)}</td>
            <td><span class="badge {statusClass(o.status)}">{o.status}</span></td>
            <td>
              {money(o.totalCents)}
              {#if o.paid}
                <span class="badge pago" style="margin-left:6px">pago</span>
              {/if}
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
            </td>
            <td>
              <div class="actions">
                <button class="btn light small" onclick={() => (editing = o)}>Editar</button>
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
</div>

{#if creating}
  <OrderModal order={null} onclose={() => (creating = false)} />
{/if}

{#if editing}
  <OrderModal order={editing} onclose={() => (editing = null)} />
{/if}

{#if printing}
  <Comprovante order={printing} onclose={() => (printing = null)} />
{/if}

{#if avisando}
  <AvisarWhatsapp order={avisando} onclose={() => (avisando = null)} />
{/if}
