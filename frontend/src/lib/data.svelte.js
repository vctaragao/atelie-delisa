import { api } from './api.js'

// Estado compartilhado entre as páginas. Fica num arquivo .svelte.js para
// poder usar runes fora de um componente.
export const data = $state({
  clients: [],
  services: [],
  orders: [],
  transactions: [],
  summary: null,
  loading: true,
  error: null,
})

/** Recarrega tudo da API. É chamado no início e após cada alteração. */
export async function refresh() {
  try {
    const [clients, services, orders, transactions, summary] = await Promise.all([
      api.listClients(),
      api.listServices(),
      api.listOrders(),
      api.listTransactions(),
      api.summary(),
    ])
    data.clients = clients
    data.services = services
    data.orders = orders
    data.transactions = transactions
    data.summary = summary
    data.error = null
  } catch (err) {
    data.error = err.message
  } finally {
    data.loading = false
  }
}

/**
 * Executa uma ação que altera dados e recarrega o estado.
 * Devolve true em caso de sucesso; em caso de erro, guarda a mensagem em
 * data.error e devolve false.
 */
export async function mutate(action) {
  try {
    await action()
    await refresh()
    return true
  } catch (err) {
    data.error = err.message
    return false
  }
}
