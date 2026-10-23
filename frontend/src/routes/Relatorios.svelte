<script>
  import { api } from '../lib/api.js'
  import {
    money,
    fmtDate,
    today,
    addDays,
    monthOf,
    shiftMonth,
    statusClass,
  } from '../lib/format.js'

  let { goto } = $props()

  const PERIODOS = [
    { id: '7', label: 'Últimos 7 dias' },
    { id: '30', label: 'Últimos 30 dias' },
    { id: 'mes', label: 'Este mês' },
    { id: 'mespassado', label: 'Mês passado' },
    { id: 'ano', label: 'Este ano' },
    { id: 'custom', label: 'Período escolhido por mim' },
  ]

  const hoje = today()

  let periodo = $state('30')
  let de = $state(addDays(hoje, -29))
  let ate = $state(hoje)

  let rel = $state(null)
  let carregando = $state(true)
  let erro = $state(null)

  /** O último dia do mês "2026-02" -> "2026-02-28". O dia 0 do mês seguinte é
      o último do mês pedido, e isso já resolve fevereiro e ano bissexto. */
  function fimDoMes(mes) {
    const [ano, m] = mes.split('-').map(Number)
    const d = new Date(ano, m, 0)
    const pad = (n) => String(n).padStart(2, '0')
    return `${d.getFullYear()}-${pad(d.getMonth() + 1)}-${pad(d.getDate())}`
  }

  function escolherPeriodo(id) {
    periodo = id
    if (id === '7') {
      de = addDays(hoje, -6)
      ate = hoje
    } else if (id === '30') {
      de = addDays(hoje, -29)
      ate = hoje
    } else if (id === 'mes') {
      de = `${monthOf(hoje)}-01`
      ate = hoje
    } else if (id === 'mespassado') {
      const mes = shiftMonth(monthOf(hoje), -1)
      de = `${mes}-01`
      ate = fimDoMes(mes)
    } else if (id === 'ano') {
      de = `${hoje.slice(0, 4)}-01-01`
      ate = hoje
    }
  }

  // Recarrega sempre que o período muda. O relatório não entra no estado
  // compartilhado das outras telas porque depende do período escolhido aqui.
  $effect(() => {
    const inicio = de
    const fim = ate
    if (!inicio || !fim || inicio > fim) return

    carregando = true
    api
      .report(inicio, fim)
      .then((r) => {
        rel = r
        erro = null
      })
      .catch((e) => {
        erro = e.message
      })
      .finally(() => {
        carregando = false
      })
  })

  const vazio = $derived(rel !== null && rel.orders === 0 && rel.expenseCents === 0)

  /** Valor em centavos como o Excel brasileiro espera: 1234,56 e sem "R$". */
  const numero = (cents) => ((Number(cents) || 0) / 100).toFixed(2).replace('.', ',')

  function exportarCsv() {
    if (!rel) return

    const linhas = [
      ['Relatório do ateliê'],
      ['Período', `${fmtDate(rel.from)} a ${fmtDate(rel.to)}`],
      [],
      ['Indicador', 'Valor'],
      ['Pedidos', rel.orders],
      ['Peças', rel.pieces],
      ['Clientes atendidas', rel.clientsServed],
      ['Valor dos pedidos', numero(rel.ordersCents)],
      ['Valor médio por pedido', numero(rel.averageCents)],
      ['Já pago', numero(rel.paidCents)],
      ['A receber', numero(rel.receivableCents)],
      ['Desconto concedido', numero(rel.discountCents)],
      ['Acréscimo de urgência', numero(rel.rushCents)],
      ['Outras entradas', numero(rel.incomeCents)],
      ['Despesas', numero(rel.expenseCents)],
      ['Resultado', numero(rel.resultCents)],
      [],
      ['Serviços mais pedidos', 'Peças', 'Pedidos', 'Valor'],
      ...rel.topServices.map((s) => [s.name, s.qty, s.orders, numero(s.totalCents)]),
      [],
      ['Clientes que mais gastaram', 'Pedidos', 'Valor'],
      ...rel.topClients.map((c) => [c.name, c.orders, numero(c.totalCents)]),
      [],
      ['Pedidos por situação', 'Quantidade'],
      ...rel.byStatus.map((s) => [s.status, s.orders]),
    ]

    // Ponto e vírgula separando as colunas e o ﻿ na frente: é assim que o
    // Excel em português abre o arquivo com os acentos certos e as colunas
    // separadas, sem passar pelo assistente de importação.
    const csv = linhas
      .map((linha) => linha.map((campo) => `"${String(campo ?? '').replace(/"/g, '""')}"`).join(';'))
      .join('\r\n')

    const blob = new Blob(['﻿' + csv], { type: 'text/csv;charset=utf-8' })
    const url = URL.createObjectURL(blob)
    const a = document.createElement('a')
    a.href = url
    a.download = `relatorio-${rel.from}-a-${rel.to}.csv`
    a.click()
    URL.revokeObjectURL(url)
  }
</script>

<div class="card">
  <div class="rel-filtro">
    <div class="form-group" style="margin:0">
      <label for="rp">Período</label>
      <select id="rp" value={periodo} onchange={(e) => escolherPeriodo(e.currentTarget.value)}>
        {#each PERIODOS as p (p.id)}
          <option value={p.id}>{p.label}</option>
        {/each}
      </select>
    </div>
    <div class="form-group" style="margin:0">
      <label for="rde">De</label>
      <input id="rde" type="date" bind:value={de} oninput={() => (periodo = 'custom')} />
    </div>
    <div class="form-group" style="margin:0">
      <label for="rate">Até</label>
      <input id="rate" type="date" bind:value={ate} oninput={() => (periodo = 'custom')} />
    </div>
    <button class="btn light" onclick={exportarCsv} disabled={!rel}>↓ Exportar CSV</button>
  </div>
  {#if de > ate}
    <p class="field-error">A data inicial está depois da final.</p>
  {/if}
</div>

{#if erro}
  <div class="notice error">{erro}</div>
{/if}

{#if carregando && !rel}
  <div class="card"><div class="empty">Montando o relatório…</div></div>
{:else if rel}
  <p class="rel-periodo">
    De <b>{fmtDate(rel.from)}</b> a <b>{fmtDate(rel.to)}</b>, contando os pedidos pela data
    de entrada. Pedidos cancelados ficam fora dos valores.
  </p>

  {#if vazio}
    <div class="card">
      <div class="empty">
        Nenhum pedido e nenhum lançamento neste período.
        <br />
        <button class="btn light small" style="margin-top:10px" onclick={() => goto('ordens')}>
          Ver as ordens de serviço
        </button>
      </div>
    </div>
  {:else}
    <div class="cards">
      <div class="card">
        <div class="metric-label">Pedidos</div>
        <div class="metric">{rel.orders}</div>
        <div class="metric-rodape">{rel.pieces} peças no total</div>
      </div>
      <div class="card">
        <div class="metric-label">Valor dos pedidos</div>
        <div class="metric">{money(rel.ordersCents)}</div>
        <div class="metric-rodape">já com desconto e urgência</div>
      </div>
      <div class="card">
        <div class="metric-label">Valor médio por pedido</div>
        <div class="metric">{money(rel.averageCents)}</div>
        <div class="metric-rodape">{rel.clientsServed} clientes atendidas</div>
      </div>
      <div class="card">
        <div class="metric-label">Resultado do período</div>
        <div class="metric" class:gold={rel.resultCents < 0}>{money(rel.resultCents)}</div>
        <div class="metric-rodape">o que entrou menos as despesas</div>
      </div>
    </div>

    <div class="cards">
      <div class="card">
        <div class="metric-label">Já pago</div>
        <div class="metric" style="color:var(--green)">{money(rel.paidCents)}</div>
      </div>
      <div class="card">
        <div class="metric-label">A receber</div>
        <div class="metric gold">{money(rel.receivableCents)}</div>
      </div>
      <div class="card">
        <div class="metric-label">Outras entradas</div>
        <div class="metric">{money(rel.incomeCents)}</div>
        <div class="metric-rodape">lançadas no financeiro</div>
      </div>
      <div class="card">
        <div class="metric-label">Despesas</div>
        <div class="metric" style="color:var(--danger)">{money(rel.expenseCents)}</div>
        <div class="metric-rodape">lançadas no financeiro</div>
      </div>
    </div>

    <div class="dashboard-grid">
      <div class="card">
        <div class="section-title">
          <h2>Serviços mais pedidos</h2>
          <span class="agenda-resumo">por valor</span>
        </div>
        <div class="table-wrap">
          <table>
            <thead>
              <tr><th style="width:34px"></th><th>Serviço</th><th>Peças</th><th>Pedidos</th><th>Valor</th></tr>
            </thead>
            <tbody>
              {#each rel.topServices as s, i (s.name)}
                <tr>
                  <td><span class="rel-posicao">{i + 1}</span></td>
                  <td style="white-space:normal"><b>{s.name}</b></td>
                  <td>{s.qty}</td>
                  <td>{s.orders}</td>
                  <td><b>{money(s.totalCents)}</b></td>
                </tr>
              {:else}
                <tr><td colspan="5" class="empty">Nenhum serviço no período.</td></tr>
              {/each}
            </tbody>
          </table>
        </div>
        <p class="rel-nota">
          O valor dos serviços é o da lista de peças, antes do desconto e da urgência —
          esses dois são do pedido inteiro.
        </p>
      </div>

      <div class="card">
        <div class="section-title">
          <h2>Clientes que mais gastaram</h2>
          <span class="agenda-resumo">as 5 primeiras</span>
        </div>
        <div class="table-wrap">
          <table>
            <thead>
              <tr><th style="width:34px"></th><th>Cliente</th><th>Pedidos</th><th>Valor</th></tr>
            </thead>
            <tbody>
              {#each rel.topClients as c, i (c.id)}
                <tr>
                  <td><span class="rel-posicao">{i + 1}</span></td>
                  <td style="white-space:normal"><b>{c.name}</b></td>
                  <td>{c.orders}</td>
                  <td><b>{money(c.totalCents)}</b></td>
                </tr>
              {:else}
                <tr><td colspan="4" class="empty">Nenhuma cliente no período.</td></tr>
              {/each}
            </tbody>
          </table>
        </div>
      </div>
    </div>

    <div class="card">
      <div class="section-title"><h2>Pedidos por situação</h2></div>
      <div class="rel-status">
        {#each rel.byStatus as s (s.status)}
          <div class="rel-status-caixa">
            <div class="rel-status-numero">{s.orders}</div>
            <span class="badge {statusClass(s.status)}">{s.status}</span>
          </div>
        {/each}
      </div>
      <p class="rel-nota">
        Aqui entram todos os pedidos do período, cancelados inclusive.
      </p>
    </div>

    <div class="card">
      <div class="section-title"><h2>Resumo do período</h2></div>
      <div class="rel-resumo">
        <div>
          <span>Desconto concedido</span>
          <b style="color:var(--green)">− {money(rel.discountCents)}</b>
        </div>
        <div>
          <span>Acréscimo de urgência</span>
          <b style="color:var(--aviso-escuro)">+ {money(rel.rushCents)}</b>
        </div>
        <div>
          <span>Peças trabalhadas</span>
          <b>{rel.pieces}</b>
        </div>
        <div>
          <span>Clientes atendidas</span>
          <b>{rel.clientsServed}</b>
        </div>
      </div>
    </div>
  {/if}
{/if}
