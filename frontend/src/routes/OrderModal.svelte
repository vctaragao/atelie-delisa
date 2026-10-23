<script>
  import Modal from '../lib/Modal.svelte'
  import { api } from '../lib/api.js'
  import { data, mutate, refresh } from '../lib/data.svelte.js'
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
    // O formulário trabalha por peça de roupa: cada peça com os serviços que
    // vão nela. No banco continua um serviço por linha, cada um guardando o
    // nome da sua peça — a montagem e o desmonte ficam nas duas funções
    // abaixo.
    pecas: agruparPorPeca(order?.items),
  })

  function servicoVazio() {
    return { serviceId: null, name: '', qty: 1, price: 0 }
  }

  /** Monta os blocos do formulário a partir dos itens gravados no pedido. */
  function agruparPorPeca(items) {
    if (!items?.length) return [{ nome: '', servicos: [servicoVazio()] }]

    const grupos = []
    for (const i of items) {
      const nome = i.piece || ''
      // Serviços da mesma peça caem no mesmo bloco, na ordem em que a peça
      // apareceu pela primeira vez.
      let grupo = grupos.find((g) => g.nome === nome)
      if (!grupo) {
        grupo = { nome, servicos: [] }
        grupos.push(grupo)
      }
      grupo.servicos.push({
        serviceId: i.serviceId,
        name: i.name,
        qty: i.qty,
        price: fromCents(i.priceCents),
      })
    }
    return grupos
  }

  /** Desmonta os blocos de volta em uma lista de serviços, cada um com a sua
      peça. A ordem das peças e dos serviços dentro delas é preservada. */
  function servicosDoFormulario() {
    const lista = []
    for (const peca of form.pecas) {
      for (const s of peca.servicos) {
        lista.push({
          serviceId: s.serviceId,
          name: (s.name || '').trim() || 'Serviço personalizado',
          piece: (peca.nome || '').trim(),
          qty: Number(s.qty) || 1,
          priceCents: toCents(s.price),
        })
      }
    }
    return lista
  }

  /* Sugestões para o campo da peça. A lista é só um atalho de digitação: o
     campo continua aceitando qualquer coisa, porque a cliente chega com
     "jaqueta de couro" e nenhuma lista fechada daria conta. */
  const PECAS_COMUNS = [
    'Calça jeans', 'Calça social', 'Vestido', 'Saia', 'Camisa', 'Blusa',
    'Blazer', 'Terno', 'Jaqueta', 'Casaco', 'Uniforme', 'Cortina',
  ]

  // As peças que ela já usou em outros pedidos vêm junto: com o tempo, a
  // lista passa a falar o vocabulário do ateliê.
  const sugestoesDePeca = $derived.by(() => {
    const usadas = new Set()
    for (const o of data.orders) {
      for (const i of o.items || []) {
        if (i.piece) usadas.add(i.piece)
      }
    }
    for (const p of PECAS_COMUNS) usadas.add(p)
    return [...usadas].sort((a, b) => a.localeCompare(b, 'pt-BR'))
  })

  let saving = $state(false)
  let formError = $state(null)

  /* Cadastro da cliente sem sair do pedido. A cliente nova chega no balcão
     junto com a peça; mandar a Maria até a tela de Clientes e voltar faria
     ela perder o que já tinha digitado aqui.
     novoCliente: null = fechado. */
  let novoCliente = $state(null)
  let salvandoCliente = $state(false)
  let erroCliente = $state(null)
  let campoNomeCliente = $state(null)

  function abrirNovoCliente() {
    erroCliente = null
    novoCliente = { name: '', phone: '', address: '' }
    // Deixa o cursor no nome, para ela já começar a digitar.
    requestAnimationFrame(() => campoNomeCliente?.focus())
  }

  async function salvarNovoCliente() {
    erroCliente = null
    const name = novoCliente.name.trim()
    if (!name) {
      erroCliente = 'Escreva o nome da cliente.'
      campoNomeCliente?.focus()
      return
    }

    salvandoCliente = true
    try {
      const criada = await api.createClient({
        name,
        phone: novoCliente.phone.trim(),
        address: novoCliente.address.trim(),
        notes: '',
      })
      await refresh()
      // Já escolhida no pedido: foi para isso que ela cadastrou aqui.
      form.clientId = criada.id
      novoCliente = null
    } catch (err) {
      erroCliente = err.message
    } finally {
      salvandoCliente = false
    }
  }

  /* Enter dentro do cadastro salva a cliente. Sem isto o Enter chegaria ao
     formulário do pedido e tentaria salvar o pedido pela metade. */
  function teclaNoCadastro(event) {
    if (event.key !== 'Enter') return
    event.preventDefault()
    salvarNovoCliente()
  }

  // A conta mostrada aqui segue a mesma regra do banco: desconto e urgência
  // sobre o subtotal dos serviços.
  const conta = $derived(
    orderAmounts(
      form.pecas.flatMap((p) =>
        p.servicos.map((s) => ({ qty: s.qty, priceCents: toCents(s.price) })),
      ),
      form.discountPercent,
      form.rushPercent,
    ),
  )

  // Prazo entre a entrada e a entrega combinada, para avisar quando o pedido
  // é de urgência.
  const prazo = $derived(prazoEmHoras(form.date, form.due, form.dueTime))
  const urgente = $derived(prazo !== null && prazo < URGENCIA_HORAS)

  function addPeca() {
    form.pecas.push({ nome: '', servicos: [servicoVazio()] })
  }

  function removePeca(index) {
    form.pecas.splice(index, 1)
    // O pedido precisa de pelo menos um serviço para existir; o formulário
    // nunca fica sem nenhuma linha para preencher.
    if (form.pecas.length === 0) addPeca()
  }

  function addServico(peca) {
    peca.servicos.push(servicoVazio())
  }

  function removeServico(peca, index) {
    peca.servicos.splice(index, 1)
    if (peca.servicos.length > 0) return

    // Peça sem serviço nenhum não tem o que fazer no pedido. Se era a única,
    // volta uma linha em branco em vez de deixar o formulário vazio.
    const i = form.pecas.indexOf(peca)
    if (form.pecas.length > 1) form.pecas.splice(i, 1)
    else peca.servicos.push(servicoVazio())
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

    // Cadastro aberto e não salvo: avisa em vez de engolir o que ela digitou.
    if (novoCliente) {
      formError = 'Salve ou cancele o cadastro da cliente antes de salvar o pedido.'
      return
    }

    if (!Number(form.clientId)) {
      formError = 'Escolha a cliente do pedido, ou cadastre uma nova.'
      return
    }

    const items = servicosDoFormulario().filter((i) => i.qty > 0)

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
        <div class="cliente-campo">
          <!-- A opção 0 é o lugar onde o <select> para quando ainda não há
               nenhuma cliente cadastrada; a validação do envio cobre ela. -->
          <select id="oc" bind:value={form.clientId}>
            <option value={0}>— escolha a cliente —</option>
            {#each data.clients as c (c.id)}
              <option value={c.id}>{c.name}</option>
            {/each}
          </select>
          {#if !novoCliente}
            <button type="button" class="btn light small" onclick={abrirNovoCliente}>
              + Cadastrar
            </button>
          {/if}
        </div>
      </div>
      <div class="form-group">
        <label for="od">Data de entrada</label>
        <input id="od" type="date" bind:value={form.date} />
      </div>
    </div>

    {#if novoCliente}
      <!-- Cadastro dentro do pedido. Não é um <form> dentro de outro — o HTML
           não permite —, por isso o Enter de cada campo é tratado à mão. -->
      <div class="novo-cliente">
        <div class="novo-cliente-titulo">
          <b>Cadastrar cliente</b>
          <button
            type="button"
            class="close"
            aria-label="Fechar o cadastro de cliente"
            onclick={() => (novoCliente = null)}
          >
            ×
          </button>
        </div>

        <div class="grid2">
          <div class="form-group">
            <label for="ncn">Nome *</label>
            <input
              id="ncn"
              bind:this={campoNomeCliente}
              bind:value={novoCliente.name}
              onkeydown={teclaNoCadastro}
            />
          </div>
          <div class="form-group">
            <label for="ncp">Telefone</label>
            <input
              id="ncp"
              bind:value={novoCliente.phone}
              onkeydown={teclaNoCadastro}
              placeholder="(35) 99999-9999"
            />
          </div>
        </div>

        <div class="form-group">
          <label for="nca">Endereço</label>
          <input id="nca" bind:value={novoCliente.address} onkeydown={teclaNoCadastro} />
        </div>

        <p class="field-dica">
          O telefone é o que permite avisar pelo WhatsApp quando a peça ficar
          pronta. Observações e o resto da ficha você completa depois, na tela
          de Clientes.
        </p>

        {#if erroCliente}
          <p class="field-error">{erroCliente}</p>
        {/if}

        <div class="novo-cliente-foot">
          <button
            type="button"
            class="btn light small"
            onclick={() => (novoCliente = null)}
          >
            Cancelar
          </button>
          <button
            type="button"
            class="btn small"
            disabled={salvandoCliente}
            onclick={salvarNovoCliente}
          >
            {salvandoCliente ? 'Salvando…' : 'Salvar cliente'}
          </button>
        </div>
      </div>
    {/if}

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
      <label for="oitems">
        Peças e serviços
        <span style="font-weight:400;color:var(--muted)">
          uma peça por bloco, com os serviços dela dentro
        </span>
      </label>

      <div class="service-list" id="oitems">
        {#each form.pecas as peca, pi}
          <div class="peca-bloco">
            <div class="peca-cabecalho">
              <input
                list="pecas-do-atelie"
                placeholder="Peça — ex.: calça jeans"
                bind:value={peca.nome}
                aria-label="Peça de roupa"
              />
              <span class="peca-contagem">
                {peca.servicos.length}
                {peca.servicos.length === 1 ? 'serviço' : 'serviços'}
              </span>
              {#if form.pecas.length > 1}
                <button
                  type="button"
                  class="close"
                  aria-label="Remover esta peça e os serviços dela"
                  onclick={() => removePeca(pi)}
                >
                  ×
                </button>
              {/if}
            </div>

            {#each peca.servicos as servico, si}
              <div class="service-item">
                <div class="service-row">
                  <!-- Os value são strings de propósito: o <select> do DOM
                       compara valores como texto. -->
                  <select
                    value={servico.serviceId === null ? '' : String(servico.serviceId)}
                    onchange={(e) => pickService(servico, e.currentTarget.value)}
                  >
                    <option value="">Serviço personalizado</option>
                    {#each data.services as s (s.id)}
                      <option value={String(s.id)}>{s.name} — {money(s.priceCents)}</option>
                    {/each}
                  </select>
                  <input
                    type="number"
                    min="1"
                    step="1"
                    bind:value={servico.qty}
                    aria-label="Quantidade"
                  />
                  <input
                    type="number"
                    min="0"
                    step="0.01"
                    bind:value={servico.price}
                    aria-label="Preço unitário"
                  />
                  <button
                    type="button"
                    class="close"
                    aria-label="Remover este serviço"
                    onclick={() => removeServico(peca, si)}
                  >
                    ×
                  </button>
                </div>
                {#if servico.serviceId === null}
                  <input
                    placeholder="Descreva o serviço"
                    bind:value={servico.name}
                    aria-label="Descrição do serviço personalizado"
                  />
                {/if}
              </div>
            {/each}

            <button type="button" class="btn light small" onclick={() => addServico(peca)}>
              + Adicionar serviço nesta peça
            </button>
          </div>
        {/each}
      </div>

      <!-- Uma lista só para todos os campos de peça: o datalist é referenciado
           pelo id e não precisa ser repetido a cada bloco. -->
      <datalist id="pecas-do-atelie">
        {#each sugestoesDePeca as p (p)}
          <option value={p}></option>
        {/each}
      </datalist>
      <button type="button" class="btn light small" style="margin-top:8px" onclick={addPeca}>
        + Adicionar outra peça
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
