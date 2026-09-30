<script>
  import Modal from '../lib/Modal.svelte'
  import { api } from '../lib/api.js'
  import { data, mutate } from '../lib/data.svelte.js'
  import { money, toCents, fmtDate, today } from '../lib/format.js'

  let adding = $state(false)
  let saving = $state(false)
  let formError = $state(null)
  let form = $state(blank())

  const s = $derived(data.summary)

  function blank() {
    return { date: today(), type: 'entrada', description: '', value: '', payment: 'Pix' }
  }

  function open() {
    form = blank()
    formError = null
    adding = true
  }

  async function save(event) {
    event.preventDefault()
    formError = null

    saving = true
    const ok = await mutate(() =>
      api.createTransaction({
        date: form.date,
        type: form.type,
        description: form.description,
        valueCents: toCents(form.value),
        payment: form.payment,
      }),
    )
    saving = false
    if (ok) {
      adding = false
    } else {
      // Mostra o erro dentro do formulário, não no aviso global do topo.
      formError = data.error
      data.error = null
    }
  }

  async function remove(tx) {
    if (!confirm('Excluir este lançamento?')) return
    await mutate(() => api.deleteTransaction(tx.id))
  }
</script>

<div class="cards">
  <div class="card">
    <div class="metric-label">Total recebido</div>
    <div class="metric">{money(s?.revenueCents)}</div>
  </div>
  <div class="card">
    <div class="metric-label">Total pendente</div>
    <div class="metric gold">{money(s?.receivableCents)}</div>
  </div>
  <div class="card">
    <div class="metric-label">Despesas</div>
    <div class="metric">{money(s?.expenseCents)}</div>
  </div>
  <div class="card">
    <div class="metric-label">Saldo</div>
    <div class="metric">{money(s?.balanceCents)}</div>
  </div>
</div>

<div class="toolbar">
  <button class="btn" onclick={open}>+ Lançamento</button>
</div>

<div class="card">
  <div class="table-wrap">
    <table>
      <thead>
        <tr><th>Data</th><th>Descrição</th><th>Tipo</th><th>Forma</th><th>Valor</th><th></th></tr>
      </thead>
      <tbody>
        {#each data.transactions as t (t.id)}
          <tr>
            <td>{fmtDate(t.date)}</td>
            <td>{t.description}</td>
            <td>
              <span class={t.type === 'entrada' ? 'money-in' : 'money-out'}>
                {t.type === 'entrada' ? 'Entrada' : 'Saída'}
              </span>
            </td>
            <td>{t.payment || '—'}</td>
            <td>{money(t.valueCents)}</td>
            <td><button class="btn danger small" onclick={() => remove(t)}>Excluir</button></td>
          </tr>
        {:else}
          <tr>
            <td colspan="6" class="empty">
              Nenhum lançamento manual. Pagamentos dos pedidos aparecem nos totais.
            </td>
          </tr>
        {/each}
      </tbody>
    </table>
  </div>
</div>

{#if adding}
  <Modal title="Novo lançamento" onclose={() => (adding = false)}>
    <form onsubmit={save}>
      <div class="grid2">
        <div class="form-group">
          <label for="td">Data *</label>
          <input id="td" type="date" bind:value={form.date} required />
        </div>
        <div class="form-group">
          <label for="tt">Tipo</label>
          <select id="tt" bind:value={form.type}>
            <option value="entrada">Entrada</option>
            <option value="saida">Saída</option>
          </select>
        </div>
      </div>
      <div class="form-group">
        <label for="tds">Descrição *</label>
        <input id="tds" bind:value={form.description} required placeholder="Ex.: Compra de linha" />
      </div>
      <div class="grid2">
        <div class="form-group">
          <label for="tv">Valor (R$) *</label>
          <input id="tv" type="number" min="0" step="0.01" bind:value={form.value} required />
        </div>
        <div class="form-group">
          <label for="tp">Forma de pagamento</label>
          <select id="tp" bind:value={form.payment}>
            <option>Pix</option>
            <option>Dinheiro</option>
            <option>Cartão</option>
            <option>Outro</option>
          </select>
        </div>
      </div>

      {#if formError}
        <p class="field-error">{formError}</p>
      {/if}

      <div class="modal-foot">
        <button type="button" class="btn light" onclick={() => (adding = false)}>Cancelar</button>
        <button class="btn" disabled={saving}>{saving ? 'Salvando…' : 'Salvar lançamento'}</button>
      </div>
    </form>
  </Modal>
{/if}
