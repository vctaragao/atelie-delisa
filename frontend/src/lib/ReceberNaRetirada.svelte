<script>
  /* A pergunta do balcão: a peça acabou de sair e o pedido tem saldo aberto.
     Recebeu agora?

     Esta é a tela que fecha o vão entre o pagamento e o lançamento. O
     dinheiro é recebido de pé, com a cliente na frente, e o lançamento só
     acontecia depois — quando ela se lembrava. É daí que vem a baixa perdida
     e, dias depois, a cobrança repetida.

     As três respostas cobrem o que acontece de verdade no ateliê:

     - pagou tudo agora: um clique na forma e a baixa está lançada, pelo valor
       que faltava. É o caso comum, e por isso é o mais rápido daqui;
     - pagou uma parte: abre a tela de pagamentos, onde ela digita o valor;
     - vai pagar depois: não lança dinheiro nenhum, e registra que a peça saiu
       devendo de propósito. O saldo continua aberto, mas deixa de ser
       confundido com um lançamento esquecido.

     Fechar sem responder é permitido: às vezes a cliente está falando com ela
     e a conta fica para depois. Nesse caso o pedido aparece na lista de
     conferência do Financeiro, que é o segundo cinto de segurança. */
  import Modal from './Modal.svelte'
  import { api } from './api.js'
  import { data, mutate } from './data.svelte.js'
  import { money, today, faltaReceber, FORMAS_DE_RECEBIMENTO } from './format.js'

  let { order, onclose, onpagamentos } = $props()

  // O pedido chega como estava quando a caixinha foi marcada. Depois de uma
  // baixa, a versão de verdade é a que voltou do servidor.
  const pedido = $derived(data.orders.find((o) => o.id === order.id) || order)
  const falta = $derived(faltaReceber(pedido))

  let salvando = $state('')
  let erro = $state(null)

  async function receber(forma) {
    erro = null
    salvando = forma
    const ok = await mutate(() =>
      api.addPayment(pedido.id, {
        date: today(),
        valueCents: falta,
        method: forma,
        notes: 'Recebido na retirada',
      }),
    )
    salvando = ''
    if (!ok) {
      erro = data.error
      data.error = null
      return
    }
    onclose()
  }

  async function pagaDepois() {
    erro = null
    salvando = 'depois'
    const ok = await mutate(() => api.setPayLater(pedido.id, today()))
    salvando = ''
    if (!ok) {
      erro = data.error
      data.error = null
      return
    }
    onclose()
  }
</script>

<Modal title="A cliente pagou?" {onclose}>
  <p class="receber-conta">
    A peça do pedido <b>#{pedido.number}</b> — {pedido.clientName} — saiu do
    ateliê e ainda faltam <b class="devendo">{money(falta)}</b>.
  </p>

  <div class="pseudo-label">Pagou tudo agora — como recebeu?</div>
  <div class="receber-formas">
    {#each FORMAS_DE_RECEBIMENTO as f (f)}
      <button class="btn" disabled={Boolean(salvando)} onclick={() => receber(f)}>
        {salvando === f ? 'Lançando…' : f}
      </button>
    {/each}
  </div>
  <p class="field-dica">
    A baixa é lançada hoje, pelos {money(falta)} que faltavam.
  </p>

  <div class="receber-outras">
    <button
      class="btn light"
      disabled={Boolean(salvando)}
      onclick={() => {
        onclose()
        onpagamentos?.(pedido)
      }}
    >
      Pagou só uma parte…
    </button>

    <!-- Não lança dinheiro: registra que o saldo aberto é combinado, e não
         uma baixa que ficou para trás. É o que tira o pedido da lista de
         conferência sem fingir que a conta foi paga. -->
    <button class="btn light" disabled={Boolean(salvando)} onclick={pagaDepois}>
      {salvando === 'depois' ? 'Anotando…' : 'Vai pagar depois'}
    </button>
  </div>

  {#if erro}
    <p class="field-error">{erro}</p>
  {/if}

  <div class="modal-foot">
    <span class="field-dica" style="flex:1;margin:0">
      Se fechar sem responder, o pedido fica na lista <b>Conferir pagamentos</b>,
      no Financeiro.
    </span>
    <button type="button" class="btn light" disabled={Boolean(salvando)} onclick={onclose}>
      Decido depois
    </button>
  </div>
</Modal>
