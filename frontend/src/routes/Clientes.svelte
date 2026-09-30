<script>
  import Modal from '../lib/Modal.svelte'
  import { api } from '../lib/api.js'
  import { data, mutate } from '../lib/data.svelte.js'

  let query = $state('')
  let editing = $state(null) // null = fechado, {} = novo, {id,...} = edição
  let saving = $state(false)
  let formError = $state(null)

  const filtered = $derived.by(() => {
    const q = query.trim().toLowerCase()
    if (!q) return data.clients
    return data.clients.filter((c) =>
      `${c.name} ${c.phone}`.toLowerCase().includes(q),
    )
  })

  function open(client) {
    formError = null
    editing = client
      ? { ...client }
      : { name: '', phone: '', address: '', notes: '' }
  }

  async function save(event) {
    event.preventDefault()
    formError = null

    const payload = {
      name: editing.name,
      phone: editing.phone || '',
      address: editing.address || '',
      notes: editing.notes || '',
    }

    saving = true
    const ok = await mutate(() =>
      editing.id ? api.updateClient(editing.id, payload) : api.createClient(payload),
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

  async function remove(client) {
    if (!confirm(`Excluir o cliente "${client.name}"?`)) return
    await mutate(() => api.deleteClient(client.id))
  }
</script>

<div class="toolbar">
  <input
    class="search"
    placeholder="Buscar cliente por nome ou telefone..."
    bind:value={query}
  />
  <button class="btn" onclick={() => open(null)}>+ Novo cliente</button>
</div>

<div class="card">
  <div class="table-wrap">
    <table>
      <thead>
        <tr><th>Nome</th><th>Telefone</th><th>Endereço</th><th>Pedidos</th><th></th></tr>
      </thead>
      <tbody>
        {#each filtered as c (c.id)}
          <tr>
            <td><b>{c.name}</b></td>
            <td>{c.phone || '—'}</td>
            <td>{c.address || '—'}</td>
            <td>{c.orderCount}</td>
            <td>
              <div class="actions">
                <button class="btn light small" onclick={() => open(c)}>Editar</button>
                <button class="btn danger small" onclick={() => remove(c)}>Excluir</button>
              </div>
            </td>
          </tr>
        {:else}
          <tr><td colspan="5" class="empty">Nenhum cliente encontrado.</td></tr>
        {/each}
      </tbody>
    </table>
  </div>
</div>

{#if editing}
  <Modal
    title={editing.id ? 'Editar cliente' : 'Novo cliente'}
    onclose={() => (editing = null)}
  >
    <form onsubmit={save}>
      <div class="grid2">
        <div class="form-group">
          <label for="cn">Nome *</label>
          <input id="cn" bind:value={editing.name} required />
        </div>
        <div class="form-group">
          <label for="cp">Telefone</label>
          <input id="cp" bind:value={editing.phone} placeholder="(35) 99999-9999" />
        </div>
      </div>
      <div class="form-group">
        <label for="ca">Endereço</label>
        <input id="ca" bind:value={editing.address} />
      </div>
      <div class="form-group">
        <label for="cno">Observações</label>
        <textarea id="cno" rows="3" bind:value={editing.notes}></textarea>
      </div>

      {#if formError}
        <p class="field-error">{formError}</p>
      {/if}

      <div class="modal-foot">
        <button type="button" class="btn light" onclick={() => (editing = null)}>Cancelar</button>
        <button class="btn" disabled={saving}>{saving ? 'Salvando…' : 'Salvar cliente'}</button>
      </div>
    </form>
  </Modal>
{/if}
