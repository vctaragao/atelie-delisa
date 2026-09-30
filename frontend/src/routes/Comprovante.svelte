<script>
  import Modal from '../lib/Modal.svelte'
  import { data } from '../lib/data.svelte.js'
  import { money, fmtDate } from '../lib/format.js'

  let { order, onclose } = $props()

  // O pedido da listagem traz só o nome do cliente. Telefone e endereço vêm
  // da ficha, que já está carregada na memória.
  const client = $derived(data.clients.find((c) => c.id === order.clientId))

  const printedAt = new Date().toLocaleDateString('pt-BR')

  // Sem "R$" nas linhas dos itens: em 48 mm de largura cada caractere conta,
  // e o cabeçalho do total já diz que são reais.
  const valor = (cents) => money(cents).replace('R$', '').trim()

  function imprimir() {
    window.print()
  }
</script>

<Modal title="Comprovante do pedido" {onclose}>
  <div class="comprovante">
    <div class="comprovante-topo">
      <strong>ATELIÊ DELISA</strong>
      <span>Comprovante de pedido</span>
    </div>

    <hr class="comprovante-sep" />

    <div class="comprovante-pedido">PEDIDO #{order.number}</div>

    <div class="comprovante-linha"><span>Cliente</span><span>{order.clientName}</span></div>
    {#if client?.phone}
      <div class="comprovante-linha"><span>Telefone</span><span>{client.phone}</span></div>
    {/if}
    {#if client?.address}
      <div class="comprovante-linha"><span>Endereço</span><span>{client.address}</span></div>
    {/if}
    <div class="comprovante-linha"><span>Entrada</span><span>{fmtDate(order.date)}</span></div>
    <div class="comprovante-linha"><span>Entrega</span><span>{fmtDate(order.due)}</span></div>
    <div class="comprovante-linha"><span>Situação</span><span>{order.status}</span></div>

    <hr class="comprovante-sep" />

    {#each order.items as item (item.id)}
      <div class="comprovante-item">
        <span class="comprovante-item-nome">{item.name}</span>
        <div class="comprovante-linha">
          <span>{item.qty} x {valor(item.priceCents)}</span>
          <span>{valor(item.qty * item.priceCents)}</span>
        </div>
      </div>
    {/each}

    <hr class="comprovante-sep" />

    <div class="comprovante-total">
      <span>TOTAL</span>
      <span>{money(order.totalCents)}</span>
    </div>

    <div class="comprovante-linha"><span>Pagamento</span><span>{order.payment}</span></div>
    <div class="comprovante-linha">
      <span>Situação</span><span>{order.paid ? 'PAGO' : 'PENDENTE'}</span>
    </div>

    {#if order.notes}
      <hr class="comprovante-sep" />
      <div class="comprovante-obs">
        <strong>Observações</strong>
        {order.notes}
      </div>
    {/if}

    <hr class="comprovante-sep" />

    <div class="comprovante-pe">
      <p>Emitido em {printedAt}</p>
      <p>Este comprovante registra o pedido acordado entre o ateliê e o cliente.</p>
      <p>NÃO É NOTA FISCAL. Não tem valor fiscal.</p>
    </div>
  </div>

  <div class="modal-foot">
    <button type="button" class="btn light" onclick={onclose}>Fechar</button>
    <button type="button" class="btn" onclick={imprimir}>Imprimir</button>
  </div>
</Modal>
