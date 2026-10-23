<script>
  import Modal from '../lib/Modal.svelte'
  import { api } from '../lib/api.js'
  import { data, mutate } from '../lib/data.svelte.js'
  import {
    money,
    toCents,
    fromCents,
    today,
    orderAmounts,
    prazoEmHoras,
    ORDER_STATUSES,
    PAYMENT_METHODS,
    URGENCIA_HORAS,
    URGENCIA_PADRAO,
  } from '../lib/format.js'

  let { order = null, onclose } = $props()

  const editing = order !== null

  // Cópia local do pedido. Os preços viram reais para caber nos inputs e
  // voltam a centavos ao salvar.
  let form = $state({
    clientId: order?.clientId ?? data.clients[0]?.id ?? 0,
    date: order?.date || today(),
    due: order?.due || '',
    dueTime: order?.dueTime || '',
    status: order?.status || ORDER_STATUSES[0],
    payment: order?.payment || PAYMENT_METHODS[0],
    paid: order?.paid ?? false,
    notes: order?.notes || '',
    discountPercent: order?.discountPercent ?? 0,
    rushPercent: order?.rushPercent ?? 0,
    items: order?.items?.length
      ? order.items.map((i) => ({
          serviceId: i.serviceId,
          name: i.name,
          qty: i.qty,
          price: fromCents(i.priceCents),
        }))
      : [{ serviceId: null, name: '', qty: 1, price: 0 }],
  })

  let saving = $state(false)
  let formError = $state(null)

  // A conta mostrada aqui segue a mesma regra do banco: desconto e urgência
  // sobre o subtotal dos serviços.
  const conta = $derived(
    orderAmounts(
      form.items.map((i) => ({ qty: i.qty, priceCents: toCents(i.price) })),
      form.discountPercent,
      form.rushPercent,
    ),
  )

  // Prazo entre a entrada e a entrega combinada, para avisar quando o pedido
  // é de urgência.
  const prazo = $derived(prazoEmHoras(form.date, form.due, form.dueTime))
  const urgente = $derived(prazo !== null && prazo < URGENCIA_HORAS)

  function addItem() {
    form.items.push({ serviceId: null, name: '', qty: 1, price: 0 })
  }

  function removeItem(index) {
    form.items.splice(index, 1)
  }

  // Ao escolher um serviço da tabela, copia nome e preço para o item. O preço
  // fica editável depois, para permitir um ajuste pontual sem mexer na tabela.
  function pickService(item, serviceId) {
    const id = serviceId === '' ? null : Number(serviceId)
    item.serviceId = id
    const svc = data.services.find((s) => s.id === id)
    if (svc) {
      item.name = svc.name
      item.price = fromCents(svc.priceCents)
    }
  }

  async function save(event) {
    event.preventDefault()
    formError = null

    const items = form.items
      .map((i) => ({
        serviceId: i.serviceId,
        name: (i.name || '').trim() || 'Serviço personalizado',
        qty: Number(i.qty) || 1,
        priceCents: toCents(i.price),
      }))
      .filter((i) => i.qty > 0)

    if (items.length === 0) {
      formError = 'Adicione pelo menos um serviço.'
      return
    }

    const payload = {
      clientId: Number(form.clientId),
      date: form.date,
      due: form.due,
      // Sem data marcada a hora não tem lugar na agenda; o backend também a
      // descarta, e limpar aqui evita salvar e ver o campo voltar vazio.
      dueTime: form.due ? form.dueTime : '',
      status: form.status,
      payment: form.payment,
      paid: form.paid,
      notes: form.notes,
      discountPercent: Number(form.discountPercent) || 0,
      rushPercent: Number(form.rushPercent) || 0,
      items,
    }

    saving = true
    const ok = await mutate(() =>
      editing ? api.updateOrder(order.id, payload) : api.createOrder(payload),
    )
    saving = false
    if (ok) {
      onclose()
    } else {
      // Mostra o erro dentro do formulário, não no aviso global do topo.
      formError = data.error
      data.error = null
    }
  }
</script>

<Modal title={editing ? `Editar pedido #${order.number}` : 'Novo pedido'} {onclose}>
  <form onsubmit={save}>
    <div class="grid2">
      <div class="form-group">
        <label for="oc">Cliente *</label>
        <select id="oc" bind:value={form.clientId} required>
          {#each data.clients as c (c.id)}
            <option value={c.id}>{c.name}</option>
          {/each}
        </select>
      </div>
      <div class="form-group">
        <label for="od">Data de entrada</label>
        <input id="od" type="date" bind:value={form.date} />
      </div>
    </div>

    <div class="grid2">
      <div class="form-group">
        <!-- O dia e a hora ficam no mesmo campo, lado a lado: a entrega é uma
             data e um horário, não duas informações separadas. -->
        <label for="odu">Entrega — dia e horário</label>
        <div class="entrega-campos">
          <input id="odu" type="date" bind:value={form.due} />
          <input
            id="oduh"
            type="time"
            bind:value={form.dueTime}
            aria-label="Horário de entrega"
          />
        </div>
        {#if form.dueTime && !form.due}
          <p class="field-dica">Marque também o dia, senão o horário não é guardado.</p>
        {/if}
      </div>
      <div class="form-group">
        <label for="ost">Status</label>
        <select id="ost" bind:value={form.status}>
          {#each ORDER_STATUSES as s (s)}
            <option value={s}>{s}</option>
          {/each}
        </select>
      </div>
    </div>

    <div class="form-group">
      <label for="oitems">Peças e serviços</label>
      <div class="service-list" id="oitems">
        {#each form.items as item, i}
          <div class="service-row">
            <!-- Os value são strings de propósito: o <select> do DOM compara
                 valores como texto. -->
            <select
              value={item.serviceId === null ? '' : String(item.serviceId)}
              onchange={(e) => pickService(item, e.currentTarget.value)}
            >
              <option value="">Serviço personalizado</option>
              {#each data.services as s (s.id)}
                <option value={String(s.id)}>{s.name} — {money(s.priceCents)}</option>
              {/each}
            </select>
            <input type="number" min="1" step="1" bind:value={item.qty} aria-label="Quantidade" />
            <input
              type="number"
              min="0"
              step="0.01"
              bind:value={item.price}
              aria-label="Preço unitário"
            />
            <button
              type="button"
              class="close"
              aria-label="Remover item"
              onclick={() => removeItem(i)}
            >
              ×
            </button>
          </div>
          {#if item.serviceId === null}
            <input
              placeholder="Descreva a peça/serviço"
              bind:value={item.name}
              aria-label="Descrição do serviço personalizado"
            />
          {/if}
        {/each}
      </div>
      <button type="button" class="btn light small" style="margin-top:8px" onclick={addItem}>
        + Adicionar serviço
      </button>
    </div>

    <div class="grid2">
      <div class="form-group">
        <label for="odesc">Desconto (%)</label>
        <input
          id="odesc"
          type="number"
          min="0"
          max="100"
          step="1"
          bind:value={form.discountPercent}
        />
      </div>
      <div class="form-group">
        <label for="ourg">Acréscimo de urgência (%)</label>
        <input
          id="ourg"
          type="number"
          min="0"
          max="100"
          step="1"
          bind:value={form.rushPercent}
        />
      </div>
    </div>

    {#if urgente}
      <div class="notice urgencia">
        Entrega em menos de {URGENCIA_HORAS} h
        {#if prazo >= 0}
          — faltam cerca de {Math.max(1, Math.round(prazo))} h.
        {:else}
          — o prazo combinado já passou.
        {/if}
        {#if Number(form.rushPercent) > 0}
          Acréscimo de urgência de {Number(form.rushPercent)}% aplicado.
        {:else}
          <button
            type="button"
            class="btn light small"
            style="margin-left:6px"
            onclick={() => (form.rushPercent = URGENCIA_PADRAO)}
          >
            Aplicar {URGENCIA_PADRAO}% de urgência
          </button>
        {/if}
      </div>
    {/if}

    <div class="grid2">
      <div class="form-group">
        <label for="opay">Forma de pagamento</label>
        <select id="opay" bind:value={form.payment}>
          {#each PAYMENT_METHODS as p (p)}
            <option value={p}>{p}</option>
          {/each}
        </select>
      </div>
      <div class="form-group">
        <label for="opaid">Pagamento</label>
        <select id="opaid" bind:value={form.paid}>
          <option value={false}>Pendente</option>
          <option value={true}>Pago</option>
        </select>
      </div>
    </div>

    <div class="form-group">
      <label for="onotes">Observações</label>
      <textarea
        id="onotes"
        rows="3"
        bind:value={form.notes}
        placeholder="Ex.: ajustar 2 cm, cliente pediu prazo especial..."
      ></textarea>
    </div>

    <div class="summary conta">
      {#if conta.discountCents || conta.rushCents}
        <div class="conta-linha">
          <span>Serviços</span>
          <span>{money(conta.itemsCents)}</span>
        </div>
      {/if}
      {#if conta.discountCents}
        <div class="conta-linha desconto">
          <span>Desconto de {Number(form.discountPercent)}%</span>
          <span>− {money(conta.discountCents)}</span>
        </div>
      {/if}
      {#if conta.rushCents}
        <div class="conta-linha acrescimo">
          <span>Urgência de {Number(form.rushPercent)}%</span>
          <span>+ {money(conta.rushCents)}</span>
        </div>
      {/if}
      <div class="conta-linha total">
        <span>Total do pedido</span>
        <span>{money(conta.totalCents)}</span>
      </div>
    </div>

    {#if formError}
      <p class="field-error">{formError}</p>
    {/if}

    <div class="modal-foot">
      <button type="button" class="btn light" onclick={onclose}>Cancelar</button>
      <button class="btn" disabled={saving}>{saving ? 'Salvando…' : 'Salvar pedido'}</button>
    </div>
  </form>
</Modal>
