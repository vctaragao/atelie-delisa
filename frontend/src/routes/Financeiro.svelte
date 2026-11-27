<script>
  import Modal from '../lib/Modal.svelte'
  import { api } from '../lib/api.js'
  import { data, mutate } from '../lib/data.svelte.js'
  import {
    money,
    toCents,
    fmtDate,
    today,
    temSaldoAberto,
    faltaReceber,
    statusClass,
  } from '../lib/format.js'

  let adding = $state(false)
  let saving = $state(false)
  let formError = $state(null)
  let form = $state(blank())

  const s = $derived(data.summary)

  /* Quem está devendo, do mais antigo para o mais novo.

     O "Total pendente" lá em cima sempre mostrou o valor, e só o valor: para
     saber de quem era aquele dinheiro, ela tinha de abrir a lista de ordens e
     conferir pedido por pedido. Esta tabela é a conta aberta — a soma dos
     valores daqui é exatamente o que o cartão mostra.

     Mais antigo primeiro porque é o que está parado há mais tempo, e é por
     ele que se começa a cobrar. */
  const pendentes = $derived(
    data.orders
      .filter(temSaldoAberto)
      .sort((a, b) => (a.date === b.date ? Number(a.number) - Number(b.number) : a.date < b.date ? -1 : 1)),
  )

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
    <div class="metric-rodape">
      {pendentes.length === 1 ? '1 pedido' : `${pendentes.length} pedidos`} a receber
    </div>
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

<div class="card">
  <div class="section-title">
    <h2>Pagamentos pendentes</h2>
    <span class="agenda-resumo">
      {#if pendentes.length}
        {pendentes.length === 1 ? '1 pedido' : `${pendentes.length} pedidos`} ·
        {money(pendentes.reduce((soma, o) => soma + faltaReceber(o), 0))} a receber
      {/if}
    </span>
  </div>

  {#if pendentes.length}
    <div class="table-wrap">
      <table>
        <thead>
          <tr>
            <th>Entrada</th><th>Pedido</th><th>Cliente</th><th>Entrega</th>
            <th>Etapa</th><th>Total</th><th>Já pago</th><th>Falta</th>
          </tr>
        </thead>
        <tbody>
          {#each pendentes as o (o.id)}
            <tr>
              <td>{fmtDate(o.date)}</td>
              <td><b>#{o.number}</b></td>
              <td style="white-space:normal">{o.clientName}</td>
              <td>{fmtDate(o.due)}</td>
              <td><span class="badge {statusClass(o.status)}">{o.status}</span></td>
              <td>{money(o.totalCents)}</td>
              <td>
                {#if o.paidCents > 0}
                  {money(o.paidCents)}
                {:else}
                  <span class="agenda-resumo">nada ainda</span>
                {/if}
              </td>
              <td><b class="devendo">{money(faltaReceber(o))}</b></td>
            </tr>
          {/each}
        </tbody>
      </table>
    </div>
    <p class="field-dica">
      Do mais antigo para o mais novo — é por ele que se começa a cobrar. A
      baixa de cada pagamento é dada no botão <b>Pagamentos</b> do pedido, na
      tela de Ordens, onde o filtro <b>A receber</b> mostra esta mesma lista.
      Pedido cancelado não entra aqui.
    </p>
  {:else}
    <p class="empty">Nenhum pagamento pendente — está tudo recebido.</p>
  {/if}
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
