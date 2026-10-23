<script>
  import Modal from './Modal.svelte'
  import { data } from './data.svelte.js'
  import {
    mensagemSugerida,
    preencherMensagem,
    numeroWhatsapp,
    linkWhatsapp,
  } from './whatsapp.js'

  let { order, onclose } = $props()

  // O telefone não vem no pedido, só na ficha da cliente — que já está
  // carregada na memória.
  const cliente = $derived(data.clients.find((c) => c.id === order.clientId))
  const numero = $derived(numeroWhatsapp(cliente?.phone))

  // A mensagem combinada com a situação do pedido já vem escolhida; as outras
  // ficam no seletor. Os textos são os que ela escreveu na aba Mensagens.
  const sugerida = mensagemSugerida(data.messages, order)

  let escolhidaId = $state(sugerida?.id ?? 0)
  let texto = $state(sugerida ? preencherMensagem(sugerida.body, order) : '')
  let copiado = $state(false)

  function trocarMensagem(id) {
    escolhidaId = Number(id)
    const m = data.messages.find((x) => x.id === escolhidaId)
    texto = m ? preencherMensagem(m.body, order) : ''
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

    {#if data.messages.length === 0}
      <div class="notice error">
        Não há nenhuma mensagem cadastrada. Escreva uma na aba <b>Mensagens</b>,
        ou use o botão de restaurar as mensagens padrão que há lá.
      </div>
    {:else}
      <div class="form-group">
        <label for="wmodelo">Mensagem</label>
        <select
          id="wmodelo"
          value={String(escolhidaId)}
          onchange={(e) => trocarMensagem(e.currentTarget.value)}
        >
          {#each data.messages as m (m.id)}
            <option value={String(m.id)}>{m.name}</option>
          {/each}
        </select>
      </div>
    {/if}

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
      <button type="button" class="btn light" onclick={copiar} disabled={!texto.trim()}>
        {copiado ? 'Copiado!' : 'Copiar texto'}
      </button>
      <button type="button" class="btn" onclick={abrir} disabled={!texto.trim()}>
        Abrir no WhatsApp
      </button>
    {/if}
  </div>
</Modal>
