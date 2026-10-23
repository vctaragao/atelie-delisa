<script>
  import Modal from '../lib/Modal.svelte'
  import { data } from '../lib/data.svelte.js'
  import { cupom } from '../lib/cupom.js'

  let { order, onclose } = $props()

  // O pedido da listagem traz só o nome do cliente. Telefone e endereço vêm
  // da ficha, que já está carregada na memória.
  const client = $derived(data.clients.find((c) => c.id === order.clientId))

  // O que aparece na tela é o mesmo texto que sai no papel, caractere por
  // caractere. Se estiver torto aqui, vai sair torto lá.
  const texto = $derived(cupom(order, client))

  function imprimir() {
    window.print()
  }
</script>

<Modal title="Comprovante do pedido" {onclose}>
  <pre class="comprovante">{texto}</pre>

  <div class="modal-foot">
    <button type="button" class="btn light" onclick={onclose}>Fechar</button>
    <button type="button" class="btn" onclick={imprimir}>Imprimir</button>
  </div>
</Modal>
