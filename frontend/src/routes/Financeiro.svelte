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

  /**
   * O movimento do caixa: os lançamentos escritos à mão e as baixas dos
   * pedidos, na mesma lista, do mais recente para o mais antigo.
   *
   * As baixas aparecem aqui porque elas são dinheiro que entrou de verdade e
   * já contam no "total recebido" — sem elas na lista, a soma lá em cima não
   * bateria com nada do que se vê embaixo. Elas não se editam daqui: quem
   * manda numa baixa é o pedido dela.
   */
  const lancamentos = $derived.by(() => {
    const dosPedidos = data.orders.flatMap((o) =>
      (o.payments || []).map((p) => ({
        chave: `pagamento-${p.id}`,
        date: p.date,
        descricao: `Pedido #${o.number} — ${o.clientName}`,
        detalhe: p.notes,
        type: 'entrada',
        valueCents: p.valueCents,
        payment: p.method,
        doPedido: true,
      })),
    )
    const manuais = data.transactions.map((t) => ({
      chave: `lancamento-${t.id}`,
      date: t.date,
      descricao: t.description,
      detalhe: '',
      type: t.type,
      valueCents: t.valueCents,
      payment: t.payment,
      doPedido: false,
      original: t,
    }))
    return [...dosPedidos, ...manuais].sort((a, b) =>
      a.date === b.date ? 0 : a.date < b.date ? 1 : -1,
    )
  })

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
        {#each lancamentos as l (l.chave)}
          <tr>
            <td>{fmtDate(l.date)}</td>
            <td style="white-space:normal">
              {l.descricao}
              {#if l.detalhe}<br /><span class="agenda-resumo">{l.detalhe}</span>{/if}
            </td>
            <td>
              <span class={l.type === 'entrada' ? 'money-in' : 'money-out'}>
                {l.type === 'entrada' ? 'Entrada' : 'Saída'}
              </span>
            </td>
            <td>{l.payment || '—'}</td>
            <td>{money(l.valueCents)}</td>
            <td>
              {#if l.doPedido}
                <span class="agenda-resumo">baixa do pedido</span>
              {:else}
                <button class="btn danger small" onclick={() => remove(l.original)}>
                  Excluir
                </button>
              {/if}
            </td>
          </tr>
        {:else}
          <tr>
            <td colspan="6" class="empty">
              Nada lançado ainda. Os pagamentos que você der baixa nos pedidos
              aparecem aqui sozinhos.
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
