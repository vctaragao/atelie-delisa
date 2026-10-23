<script>
  // Importação de uma planilha CSV, usada tanto pela tela de Clientes quanto
  // pela de Serviços. O que muda entre as duas está no mapa `tipos`.
  import Modal from './Modal.svelte'
  import { api } from './api.js'
  import { refresh } from './data.svelte.js'

  let { tipo, onclose } = $props()

  const tipos = {
    clientes: {
      title: 'Importar clientes de uma planilha',
      plural: 'clientes',
      colunas: 'nome, telefone, endereço, observações',
      obrigatoria: 'nome',
      modelo:
        'nome;telefone;endereco;observacoes\r\n' +
        'Maria Souza;(35) 99999-1234;Rua das Flores, 120;Cliente antiga\r\n' +
        'Joana Lima;(35) 98888-5678;Av. Central, 45;\r\n',
      arquivoModelo: 'modelo-clientes.csv',
      enviar: (file, atualizar) => api.importClients(file, atualizar),
      jaExiste: 'clientes que já estão cadastrados',
    },
    servicos: {
      title: 'Importar serviços de uma planilha',
      plural: 'serviços',
      colunas: 'nome, categoria, preço, tempo',
      obrigatoria: 'nome',
      modelo:
        'nome;categoria;preco;tempo\r\n' +
        'Bainha de calça;Ajustes;25,00;30 min\r\n' +
        'Troca de zíper;Consertos;35,00;40 min\r\n',
      arquivoModelo: 'modelo-servicos.csv',
      enviar: (file, atualizar) => api.importServices(file, atualizar),
      jaExiste: 'serviços que já estão na tabela de preços',
    },
  }

  const cfg = tipos[tipo]

  let arquivo = $state(null)
  let atualizar = $state(false)
  let enviando = $state(false)
  let erro = $state(null)
  let relatorio = $state(null)
  let fileInput

  function escolher(event) {
    arquivo = event.target.files?.[0] || null
    erro = null
    relatorio = null
  }

  function baixarModelo() {
    // O ﻿ é o que faz o Excel abrir o arquivo com os acentos certos.
    const blob = new Blob(['﻿' + cfg.modelo], { type: 'text/csv;charset=utf-8' })
    const url = URL.createObjectURL(blob)
    const a = document.createElement('a')
    a.href = url
    a.download = cfg.arquivoModelo
    a.click()
    URL.revokeObjectURL(url)
  }

  async function importar() {
    if (!arquivo) return
    erro = null
    relatorio = null
    enviando = true
    try {
      relatorio = await cfg.enviar(arquivo, atualizar)
      await refresh()
    } catch (err) {
      erro = err.message
    } finally {
      enviando = false
    }
  }
</script>

<Modal title={cfg.title} {onclose}>
  {#if relatorio}
    <div class="notice">
      <b>Pronto.</b>
      {relatorio.created}
      {relatorio.created === 1 ? 'linha adicionada' : 'linhas adicionadas'}{#if relatorio.updated}, {relatorio.updated}
        {relatorio.updated === 1 ? 'atualizada' : 'atualizadas'}{/if}{#if relatorio.skipped}, {relatorio.skipped}
        {relatorio.skipped === 1 ? 'não importada' : 'não importadas'}{/if}.
    </div>

    {#each relatorio.notes || [] as nota}
      <div class="notice">{nota}</div>
    {/each}

    {#if relatorio.problems?.length}
      <p style="font-size:13px;color:var(--muted);margin:0 0 8px">
        O que ficou de fora, linha por linha:
      </p>
      <div class="table-wrap" style="max-height:260px;overflow:auto">
        <table>
          <thead><tr><th style="width:80px">Linha</th><th>Motivo</th></tr></thead>
          <tbody>
            {#each relatorio.problems as p}
              <tr><td>{p.line}</td><td>{p.message}</td></tr>
            {/each}
          </tbody>
        </table>
      </div>
      {#if relatorio.omitted}
        <p style="font-size:12px;color:var(--muted);margin-top:8px">
          E outras {relatorio.omitted} linhas no mesmo caso.
        </p>
      {/if}
    {/if}

    <div class="modal-foot">
      <button type="button" class="btn" onclick={onclose}>Fechar</button>
    </div>
  {:else}
    <div class="notice">
      A planilha precisa ser salva como <b>CSV</b> e ter uma primeira linha com
      os títulos das colunas: <b>{cfg.colunas}</b>. Só a coluna
      <b>{cfg.obrigatoria}</b> é obrigatória; as outras podem faltar ou vir
      vazias. Ponto e vírgula ou vírgula separando as colunas, tanto faz.
    </div>

    <div class="form-group">
      <label for="arq">Arquivo CSV</label>
      <div style="display:flex;gap:8px;flex-wrap:wrap;align-items:center">
        <button type="button" class="btn light" onclick={() => fileInput.click()}>
          Escolher arquivo…
        </button>
        <span style="font-size:13px;color:var(--muted)">
          {arquivo ? arquivo.name : 'Nenhum arquivo escolhido'}
        </span>
        <input
          id="arq"
          type="file"
          accept=".csv,.txt,text/csv"
          style="display:none"
          bind:this={fileInput}
          onchange={escolher}
        />
      </div>
    </div>

    <label style="display:flex;gap:8px;align-items:flex-start;font-weight:400;font-size:13px">
      <input type="checkbox" bind:checked={atualizar} style="width:auto;margin-top:2px" />
      <span>
        Atualizar os {cfg.jaExiste}. Desmarcado, eles são deixados exatamente
        como estão e só as linhas novas entram.
      </span>
    </label>

    {#if erro}
      <p class="field-error">{erro}</p>
    {/if}

    <div class="modal-foot">
      <button type="button" class="btn light" onclick={baixarModelo}>↓ Baixar modelo</button>
      <button type="button" class="btn light" onclick={onclose}>Cancelar</button>
      <button type="button" class="btn" disabled={!arquivo || enviando} onclick={importar}>
        {enviando ? 'Importando…' : `Importar ${cfg.plural}`}
      </button>
    </div>
  {/if}
</Modal>
