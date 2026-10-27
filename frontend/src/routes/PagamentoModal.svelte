<script>
  import Modal from '../lib/Modal.svelte'
  import { api } from '../lib/api.js'
  import { data, mutate } from '../lib/data.svelte.js'
  import {
    money,
    toCents,
    fromCents,
    fmtDate,
    today,
    faltaReceber,
    FORMAS_DE_RECEBIMENTO,
  } from '../lib/format.js'

  let { order, onclose } = $props()

  // O pedido chega como estava quando a tela abriu. Depois de cada baixa, a
  // lista recarrega do servidor e é dela que vem a versão atual — sem isto, a
  // tela continuaria mostrando o saldo de antes do pagamento.
  const pedido = $derived(data.orders.find((o) => o.id === order.id) || order)
  const falta = $derived(faltaReceber(pedido))

  // O valor já vem preenchido com o que falta receber: o caso comum é a
  // cliente quitar o pedido, e aí é só confirmar. Para um sinal, ela apaga e
  // digita o que recebeu.
  const valorSugerido = () => {
    const resto = faltaReceber(pedido)
    return resto > 0 ? fromCents(resto) : ''
  }

  let form = $state({ date: today(), value: valorSugerido(), method: 'Pix', notes: '' })
  let saving = $state(false)
  let formError = $state(null)

  async function salvar(event) {
    event.preventDefault()
    formError = null

    const valor = toCents(form.value)
    if (valor <= 0) {
      formError = 'Escreva quanto a cliente pagou.'
      return
    }
    if (!form.date) {
      formError = 'Escolha a data em que o pagamento foi recebido.'
      return
    }

    saving = true
    const ok = await mutate(() =>
      api.addPayment(pedido.id, {
        date: form.date,
        valueCents: valor,
        method: form.method,
        notes: form.notes.trim(),
      }),
    )
    saving = false

    if (!ok) {
      formError = data.error
      data.error = null
      return
    }
    // A tela fica aberta: ela confere o novo saldo e, se for o caso, lança
    // outra parcela sem ter de abrir tudo de novo.
    form = { date: today(), value: valorSugerido(), method: form.method, notes: '' }
  }

  async function excluir(pagamento) {
    if (!confirm(`Excluir a baixa de ${money(pagamento.valueCents)}?`)) return
    await mutate(() => api.deletePayment(pedido.id, pagamento.id))
  }
</script>

<Modal title={`Pagamentos do pedido #${pedido.number}`} {onclose}>
  <div class="pagamento-conta">
    <div>
      <span>Total do pedido</span>
      <strong>{money(pedido.totalCents)}</strong>
    </div>
    <div>
      <span>Já pago</span>
      <strong class="recebido">{money(pedido.paidCents)}</strong>
    </div>
    <div>
      <span>Falta receber</span>
      <strong class:devendo={falta > 0}>{money(falta)}</strong>
    </div>
  </div>

  {#if pedido.balanceCents < 0}
    <p class="field-dica">
      A cliente pagou {money(-pedido.balanceCents)} a mais do que o total do pedido.
    </p>
  {/if}

  <div class="section-title" style="margin-top:18px">
    <h2>Baixas lançadas</h2>
  </div>

  {#if pedido.payments.length}
    <div class="table-wrap">
      <table>
        <thead>
          <tr><th>Data</th><th>Forma</th><th>Observação</th><th>Valor</th><th></th></tr>
        </thead>
        <tbody>
          {#each pedido.payments as p (p.id)}
            <tr>
              <td>{fmtDate(p.date)}</td>
              <td>{p.method || '—'}</td>
              <td style="white-space:normal">{p.notes || '—'}</td>
              <td><b class="recebido">{money(p.valueCents)}</b></td>
              <td>
                <button class="btn danger small" onclick={() => excluir(p)}>Excluir</button>
              </td>
            </tr>
          {/each}
        </tbody>
      </table>
    </div>
  {:else}
    <div class="empty">
      Nenhum pagamento recebido neste pedido até agora.
    </div>
  {/if}

  <form onsubmit={salvar} style="margin-top:18px">
    <div class="section-title">
      <h2>Registrar um pagamento</h2>
    </div>

    <div class="grid2">
      <div class="form-group">
        <label for="pgv">Valor recebido (R$) *</label>
        <input id="pgv" type="number" min="0" step="0.01" bind:value={form.value} />
      </div>
      <div class="form-group">
        <label for="pgd">Data do pagamento *</label>
        <input id="pgd" type="date" bind:value={form.date} />
      </div>
    </div>

    <div class="grid2">
      <div class="form-group">
        <label for="pgf">Forma de pagamento</label>
        <select id="pgf" bind:value={form.method}>
          {#each FORMAS_DE_RECEBIMENTO as f (f)}
            <option value={f}>{f}</option>
          {/each}
        </select>
      </div>
      <div class="form-group">
        <label for="pgo">Observação</label>
        <input id="pgo" bind:value={form.notes} placeholder="Ex.: sinal da entrada" />
      </div>
    </div>

    {#if formError}
      <p class="field-error">{formError}</p>
    {/if}

    <div class="modal-foot">
      <button type="button" class="btn light" onclick={onclose}>Fechar</button>
      <button class="btn" disabled={saving}>
        {saving ? 'Registrando…' : 'Registrar pagamento'}
      </button>
    </div>
  </form>
</Modal>
