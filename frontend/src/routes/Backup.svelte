<script>
  import { api } from '../lib/api.js'
  import { data, mutate } from '../lib/data.svelte.js'
  import { today } from '../lib/format.js'

  let busy = $state(false)
  let message = $state(null)
  let fileInput

  async function exportBackup() {
    busy = true
    message = null
    try {
      const payload = await api.exportBackup()
      const blob = new Blob([JSON.stringify(payload, null, 2)], { type: 'application/json' })
      const url = URL.createObjectURL(blob)
      const a = document.createElement('a')
      a.href = url
      a.download = `backup-atelie-delisa-${today()}.json`
      a.click()
      URL.revokeObjectURL(url)
      message = 'Backup exportado.'
    } catch (err) {
      data.error = err.message
    } finally {
      busy = false
    }
  }

  async function importBackup(event) {
    const file = event.target.files?.[0]
    event.target.value = '' // permite reenviar o mesmo arquivo
    if (!file) return

    message = null
    let parsed
    try {
      parsed = JSON.parse(await file.text())
    } catch {
      data.error = 'Arquivo de backup inválido: não é um JSON válido.'
      return
    }
    if (!parsed.clients || !parsed.services || !parsed.orders) {
      data.error = 'Arquivo de backup inválido: faltam clientes, serviços ou pedidos.'
      return
    }
    if (!confirm('Restaurar este backup substituirá TODOS os dados atuais. Continuar?')) return

    busy = true
    const ok = await mutate(() => api.restoreBackup(parsed))
    busy = false
    if (ok) message = 'Backup restaurado com sucesso.'
  }

  async function reset() {
    const warning =
      'ATENÇÃO: isso apagará todos os clientes, pedidos e lançamentos.\n\n' +
      'Faça um backup antes. Continuar?'
    if (!confirm(warning)) return
    if (!confirm('Confirma? Esta ação não pode ser desfeita.')) return

    busy = true
    message = null
    const ok = await mutate(() => api.resetData())
    busy = false
    if (ok) message = 'Todos os dados foram apagados.'
  }
</script>

<div class="card">
  <div class="section-title"><h2>Backup e segurança dos dados</h2></div>

  <div class="notice">
    Os dados ficam no banco SQLite do container do backend, num volume Docker
    persistente. Faça backups regularmente — principalmente antes de atualizar
    o sistema ou trocar de computador.
  </div>

  {#if message}
    <div class="notice">{message}</div>
  {/if}

  <div style="display:flex;gap:10px;flex-wrap:wrap">
    <button class="btn" onclick={exportBackup} disabled={busy}>↓ Exportar backup</button>
    <button class="btn light" onclick={() => fileInput.click()} disabled={busy}>
      ↑ Restaurar backup
    </button>
    <input
      type="file"
      accept=".json,application/json"
      style="display:none"
      bind:this={fileInput}
      onchange={importBackup}
    />
    <button class="btn danger" onclick={reset} disabled={busy}>Apagar todos os dados</button>
  </div>

  <hr style="border:0;border-top:1px solid var(--border);margin:25px 0" />

  <p style="font-size:13px;color:var(--muted);line-height:1.6">
    O arquivo exportado é um JSON com todos os clientes, serviços, pedidos e
    lançamentos. Guarde-o num pendrive, no Google Drive ou em outro local
    seguro. Para migrar de máquina, basta exportar aqui e restaurar no
    sistema novo.
  </p>
</div>
