<script>
  import Modal from '../lib/Modal.svelte'
  import { api } from '../lib/api.js'
  import { data, mutate } from '../lib/data.svelte.js'
  import { money, toCents, fromCents } from '../lib/format.js'

  let editing = $state(null)
  let saving = $state(false)
  let formError = $state(null)

  function open(service) {
    formError = null
    editing = service
      ? { ...service, price: fromCents(service.priceCents) }
      : { name: '', category: '', price: '', time: '' }
  }

  async function save(event) {
    event.preventDefault()
    formError = null

    const payload = {
      name: editing.name,
      category: editing.category || '',
      priceCents: toCents(editing.price),
      time: editing.time || '',
    }

    saving = true
    const ok = await mutate(() =>
      editing.id ? api.updateService(editing.id, payload) : api.createService(payload),
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

  async function remove(service) {
    const msg =
      `Excluir "${service.name}" da tabela de preços?\n\n` +
      'Pedidos antigos não serão alterados: eles guardam nome e preço próprios.'
    if (!confirm(msg)) return
    await mutate(() => api.deleteService(service.id))
  }
</script>

<div class="toolbar">
  <div style="flex:1">
    <p style="margin:0;color:var(--muted);font-size:13px">
      Cadastre seus serviços e preços para agilizar os pedidos.
    </p>
  </div>
  <button class="btn" onclick={() => open(null)}>+ Novo serviço</button>
</div>

<div class="card">
  <div class="table-wrap">
    <table>
      <thead>
        <tr><th>Serviço</th><th>Categoria</th><th>Preço padrão</th><th>Tempo</th><th></th></tr>
      </thead>
      <tbody>
        {#each data.services as s (s.id)}
          <tr>
            <td><b>{s.name}</b></td>
            <td>{s.category || '—'}</td>
            <td>{money(s.priceCents)}</td>
            <td>{s.time || '—'}</td>
            <td>
              <div class="actions">
                <button class="btn light small" onclick={() => open(s)}>Editar</button>
                <button class="btn danger small" onclick={() => remove(s)}>Excluir</button>
              </div>
            </td>
          </tr>
        {:else}
          <tr><td colspan="5" class="empty">Nenhum serviço cadastrado.</td></tr>
        {/each}
      </tbody>
    </table>
  </div>
</div>

{#if editing}
  <Modal title={editing.id ? 'Editar serviço' : 'Novo serviço'} onclose={() => (editing = null)}>
    <form onsubmit={save}>
      <div class="form-group">
        <label for="sn">Nome do serviço *</label>
        <input id="sn" bind:value={editing.name} required placeholder="Ex.: Bainha de calça" />
      </div>
      <div class="grid3">
        <div class="form-group">
          <label for="sp">Preço padrão (R$) *</label>
          <input id="sp" type="number" min="0" step="0.01" bind:value={editing.price} required />
        </div>
        <div class="form-group">
          <label for="st">Tempo estimado</label>
          <input id="st" bind:value={editing.time} placeholder="Ex.: 30 min" />
        </div>
        <div class="form-group">
          <label for="sc">Categoria</label>
          <input id="sc" bind:value={editing.category} placeholder="Ex.: Ajustes" />
        </div>
      </div>

      {#if formError}
        <p class="field-error">{formError}</p>
      {/if}

      <div class="modal-foot">
        <button type="button" class="btn light" onclick={() => (editing = null)}>Cancelar</button>
        <button class="btn" disabled={saving}>{saving ? 'Salvando…' : 'Salvar'}</button>
      </div>
    </form>
  </Modal>
{/if}
