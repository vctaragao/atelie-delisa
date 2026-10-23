import { api } from './api.js'

// Estado compartilhado entre as páginas. Fica num arquivo .svelte.js para
// poder usar runes fora de um componente.
export const data = $state({
  clients: [],
  services: [],
  orders: [],
  transactions: [],
  messages: [],
  summary: null,
  loading: true,
  error: null,
})

/** Recarrega tudo da API. É chamado no início e após cada alteração. */
export async function refresh() {
  try {
    const [clients, services, orders, transactions, messages, summary] = await Promise.all([
      api.listClients(),
      api.listServices(),
      api.listOrders(),
      api.listTransactions(),
      api.listMessages(),
      api.summary(),
    ])
    data.clients = clients
    // A ordem alfabética dos serviços é aplicada aqui, uma vez só, para valer
    // em todo lugar onde eles aparecem: a tabela de preços e o seletor do
    // pedido. O banco já devolve ordenado, mas o COLLATE NOCASE do SQLite não
    // conhece acentos — "Órfão" iria para depois do Z. O localeCompare em
    // pt-BR trata o acento como a letra sem acento, que é o esperado.
    data.services = [...services].sort((a, b) => a.name.localeCompare(b.name, 'pt-BR'))
    data.orders = orders
    data.transactions = transactions
    data.messages = messages
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
