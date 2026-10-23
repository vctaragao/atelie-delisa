<script>
  import Modal from '../lib/Modal.svelte'
  import { api } from '../lib/api.js'
  import { data, mutate } from '../lib/data.svelte.js'
  import { ORDER_STATUSES } from '../lib/format.js'
  import { APELIDOS, preencherMensagem, PEDIDO_DE_EXEMPLO } from '../lib/whatsapp.js'

  let editing = $state(null)
  let saving = $state(false)
  let formError = $state(null)

  // A caixa de texto do formulário, para inserir o apelido onde o cursor está.
  let caixaTexto = $state(null)

  function open(mensagem) {
    formError = null
    editing = mensagem
      ? { ...mensagem }
      : { name: '', body: '', suggestStatus: '' }
  }

  /** Enfia o apelido na posição do cursor, em vez de jogá-lo no fim: ela
      costuma querer o nome da cliente no meio da frase. */
  function inserirApelido(chave) {
    const campo = caixaTexto
    if (!campo) {
      editing.body += chave
      return
    }
    const inicio = campo.selectionStart ?? editing.body.length
    const fim = campo.selectionEnd ?? inicio
    editing.body = editing.body.slice(0, inicio) + chave + editing.body.slice(fim)

    // Devolve o foco com o cursor depois do apelido, para ela continuar
    // escrevendo de onde parou.
    const depois = inicio + chave.length
    requestAnimationFrame(() => {
      campo.focus()
      campo.setSelectionRange(depois, depois)
    })
  }

  const previa = $derived(editing ? preencherMensagem(editing.body, PEDIDO_DE_EXEMPLO) : '')

  async function save(event) {
    event.preventDefault()
    formError = null

    const payload = {
      name: editing.name,
      body: editing.body,
      suggestStatus: editing.suggestStatus || '',
    }

    saving = true
    const ok = await mutate(() =>
      editing.id ? api.updateMessage(editing.id, payload) : api.createMessage(payload),
    )
    saving = false
    if (ok) {
      editing = null
    } else {
      // Mostra o erro dentro do formulário, não no aviso global do topo.
      formError = data.error
      data.error = null
    }
  }

  async function remove(mensagem) {
    if (!confirm(`Excluir a mensagem "${mensagem.name}"?`)) return
    await mutate(() => api.deleteMessage(mensagem.id))
  }

  async function restaurar() {
    const msg =
      'Restaurar as três mensagens padrão?\n\n' +
      'As mensagens de agora serão apagadas, inclusive as que você escreveu.'
    if (!confirm(msg)) return
    await mutate(() => api.restoreMessages())
  }
</script>

<div class="toolbar">
  <div style="flex:1">
    <p style="margin:0;color:var(--muted);font-size:13px">
      Os textos que o botão <b>Avisar</b> dos pedidos oferece. Escreva com as suas
      palavras — o sistema só troca os apelidos entre chaves pelos dados do pedido.
    </p>
  </div>
  <button class="btn light" onclick={restaurar}>↺ Restaurar padrão</button>
  <button class="btn" onclick={() => open(null)}>+ Nova mensagem</button>
</div>

{#each data.messages as m (m.id)}
  <div class="card msg-card">
    <div class="section-title">
      <h2>{m.name}</h2>
      <div class="actions">
        <button class="btn light small" onclick={() => open(m)}>Editar</button>
        <button class="btn danger small" onclick={() => remove(m)}>Excluir</button>
      </div>
    </div>

    <p class="msg-quando">
      {#if m.suggestStatus}
        Já vem escolhida quando o pedido está em
        <b>{m.suggestStatus}</b>.
      {:else}
        Não é sugerida sozinha; aparece na lista do aviso.
      {/if}
    </p>

    <pre class="msg-corpo">{m.body}</pre>

    <details class="msg-previa">
      <summary>Ver preenchida com um pedido de exemplo</summary>
      <pre class="msg-corpo">{preencherMensagem(m.body, PEDIDO_DE_EXEMPLO)}</pre>
    </details>
  </div>
{:else}
  <div class="card">
    <div class="empty">
      Nenhuma mensagem cadastrada. Clique em "Restaurar padrão" para voltar com
      as três do começo, ou em "Nova mensagem" para escrever a sua.
    </div>
  </div>
{/each}

{#if editing}
  <Modal
    title={editing.id ? 'Editar mensagem' : 'Nova mensagem'}
    onclose={() => (editing = null)}
  >
    <form onsubmit={save}>
      <div class="grid2">
        <div class="form-group">
          <label for="mn">Nome da mensagem *</label>
          <input
            id="mn"
            bind:value={editing.name}
            required
            placeholder="Ex.: Serviço pronto para retirada"
          />
        </div>
        <div class="form-group">
          <label for="ms">Sugerir quando o pedido estiver em</label>
          <select id="ms" bind:value={editing.suggestStatus}>
            <option value="">Não sugerir sozinha</option>
            {#each ORDER_STATUSES as s (s)}
              <option value={s}>{s}</option>
            {/each}
          </select>
        </div>
      </div>

      <div class="form-group">
        <label for="mb">Texto da mensagem *</label>
        <textarea
          id="mb"
          rows="8"
          bind:this={caixaTexto}
          bind:value={editing.body}
          required
          placeholder="Oi, {'{cliente}'}! Aqui é do Ateliê Delisa."
        ></textarea>
      </div>

      <div class="form-group">
        <label for="mapelidos">Apelidos — clique para inserir no texto</label>
        <div class="apelidos" id="mapelidos">
          {#each APELIDOS as a (a.chave)}
            <button
              type="button"
              class="apelido"
              title={a.descricao}
              onclick={() => inserirApelido(a.chave)}
            >
              {a.chave}
            </button>
          {/each}
        </div>
        <p class="rel-nota">
          {#each APELIDOS as a, i (a.chave)}{i > 0 ? ' · ' : ''}<b>{a.chave}</b>
            {a.descricao}{/each}
        </p>
      </div>

      <div class="form-group">
        <label for="mprevia">Como vai ficar</label>
        <pre class="msg-corpo" id="mprevia">{previa || '—'}</pre>
      </div>

      {#if formError}
        <p class="field-error">{formError}</p>
      {/if}

      <div class="modal-foot">
        <button type="button" class="btn light" onclick={() => (editing = null)}>Cancelar</button>
        <button class="btn" disabled={saving}>{saving ? 'Salvando…' : 'Salvar mensagem'}</button>
      </div>
    </form>
  </Modal>
{/if}
