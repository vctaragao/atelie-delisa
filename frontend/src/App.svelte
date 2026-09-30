<script>
  import { data, refresh } from './lib/data.svelte.js'
  import Dashboard from './routes/Dashboard.svelte'
  import Clientes from './routes/Clientes.svelte'
  import Ordens from './routes/Ordens.svelte'
  import Financeiro from './routes/Financeiro.svelte'
  import Servicos from './routes/Servicos.svelte'
  import Backup from './routes/Backup.svelte'
  import OrderModal from './routes/OrderModal.svelte'

  const pages = [
    { id: 'dashboard', icon: '⌂', label: 'Dashboard', title: 'Dashboard', subtitle: 'Visão geral do seu ateliê', component: Dashboard },
    { id: 'clientes', icon: '♙', label: 'Clientes', title: 'Clientes', subtitle: 'Cadastro e histórico', component: Clientes },
    { id: 'ordens', icon: '✂', label: 'Ordens de serviço', title: 'Ordens de serviço', subtitle: 'Acompanhe cada peça do início à entrega', component: Ordens },
    { id: 'financeiro', icon: 'R$', label: 'Financeiro', title: 'Financeiro', subtitle: 'Entradas, saídas e valores a receber', component: Financeiro },
    { id: 'servicos', icon: '☷', label: 'Serviços', title: 'Serviços', subtitle: 'Sua tabela de preços', component: Servicos },
    { id: 'backup', icon: '↕', label: 'Backup', title: 'Backup', subtitle: 'Proteja os dados do seu ateliê', component: Backup },
  ]

  // O Vite define DEV como true em `vite dev` e false em `vite build`.
  // É o que separa esta cópia de teste do sistema que o ateliê usa.
  const ambienteDeTeste = import.meta.env.DEV

  // Título absoluto, e não um prefixo: com hot reload isto roda de novo, e
  // um prefixo acabaria repetido na aba.
  if (ambienteDeTeste) document.title = 'TESTE — Ateliê Delisa'

  let current = $state('dashboard')
  let newOrder = $state(false)

  const page = $derived(pages.find((p) => p.id === current))

  refresh()

  function goto(id) {
    current = id
  }

  function openNewOrder() {
    if (data.clients.length === 0) {
      data.error = 'Cadastre primeiro um cliente para poder criar um pedido.'
      current = 'clientes'
      return
    }
    newOrder = true
  }
</script>

{#if ambienteDeTeste}
  <div class="faixa-teste" aria-hidden="true"></div>
{/if}

<div class="app">
  <aside class="sidebar">
    <div class="logo">
      <small>Gestão do</small>
      <strong>Ateliê <span>Delisa</span></strong>
      {#if ambienteDeTeste}
        <span
          class="selo-teste"
          title="Você está no ambiente de teste. Nada aqui altera os dados reais do ateliê."
        >
          Ambiente de teste
        </span>
      {/if}
    </div>
    <nav class="nav">
      {#each pages as p (p.id)}
        <button class:active={current === p.id} onclick={() => goto(p.id)}>
          <span>{p.icon}</span> {p.label}
        </button>
      {/each}
    </nav>
    <div class="sidebar-bottom">
      Backend Go + SQLite<br />Os dados ficam no servidor.
    </div>
  </aside>

  <main class="main">
    <header class="topbar">
      <div>
        <h1>{page.title}</h1>
        <p>{page.subtitle}</p>
      </div>
      <button class="btn gold" onclick={openNewOrder}>+ Novo pedido</button>
    </header>

    {#if data.error}
      <div class="notice error">
        {data.error}
        <button
          class="btn light small"
          style="margin-left:10px"
          onclick={() => { data.error = null; refresh() }}
        >
          Tentar de novo
        </button>
      </div>
    {/if}

    {#if data.loading}
      <div class="card"><div class="empty">Carregando…</div></div>
    {:else}
      {@const Page = page.component}
      <Page {goto} />
    {/if}
  </main>
</div>

{#if newOrder}
  <OrderModal order={null} onclose={() => (newOrder = false)} />
{/if}
