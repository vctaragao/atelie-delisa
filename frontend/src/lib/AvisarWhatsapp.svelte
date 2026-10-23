<script>
  import Modal from './Modal.svelte'
  import { data } from './data.svelte.js'
  import { MODELOS, modeloSugerido, numeroWhatsapp, linkWhatsapp } from './whatsapp.js'

  let { order, onclose } = $props()

  // O telefone não vem no pedido, só na ficha da cliente — que já está
  // carregada na memória.
  const cliente = $derived(data.clients.find((c) => c.id === order.clientId))
  const numero = $derived(numeroWhatsapp(cliente?.phone))

  let modelo = $state(modeloSugerido(order))
  let texto = $state(MODELOS.find((m) => m.id === modeloSugerido(order)).texto(order))
  let copiado = $state(false)

  function trocarModelo(id) {
    modelo = id
    texto = MODELOS.find((m) => m.id === id).texto(order)
    copiado = false
  }

  function abrir() {
    // Aba nova: a janela do ateliê continua aberta atrás, com o pedido na tela.
    window.open(linkWhatsapp(numero, texto), '_blank', 'noopener')
    onclose()
  }

  async function copiar() {
    try {
      await navigator.clipboard.writeText(texto)
      copiado = true
    } catch {
      copiado = false
    }
  }
</script>

<Modal title={`Avisar ${order.clientName} pelo WhatsApp`} {onclose}>
  {#if !cliente}
    <div class="notice error">
      A ficha desta cliente não está mais cadastrada, então não há telefone para
      onde mandar.
    </div>
  {:else if !numero}
    <div class="notice error">
      {#if cliente.phone}
        O telefone <b>{cliente.phone}</b> está incompleto — falta o DDD. Corrija
        na tela de Clientes e o aviso fica disponível.
      {:else}
        Esta cliente está sem telefone na ficha. Cadastre o número na tela de
        Clientes para poder avisar por aqui.
      {/if}
    </div>
  {:else}
    <div class="notice">
      O WhatsApp abre numa aba nova com a mensagem já escrita, no número
      <b>{cliente.phone}</b>. Quem aperta enviar é você — nada sai daqui sozinho.
    </div>

    <div class="form-group">
      <label for="wmodelo">Mensagem</label>
      <select id="wmodelo" value={modelo} onchange={(e) => trocarModelo(e.currentTarget.value)}>
        {#each MODELOS as m (m.id)}
          <option value={m.id}>{m.label}</option>
        {/each}
      </select>
    </div>

    <div class="form-group">
      <label for="wtexto">
        Texto
        <span style="font-weight:400;color:var(--muted)">pode editar antes de enviar</span>
      </label>
      <textarea id="wtexto" rows="9" bind:value={texto}></textarea>
    </div>
  {/if}

  <div class="modal-foot">
    <button type="button" class="btn light" onclick={onclose}>Fechar</button>
    {#if numero}
      <button type="button" class="btn light" onclick={copiar}>
        {copiado ? 'Copiado!' : 'Copiar texto'}
      </button>
      <button type="button" class="btn" onclick={abrir} disabled={!texto.trim()}>
        Abrir no WhatsApp
      </button>
    {/if}
  </div>
</Modal>
