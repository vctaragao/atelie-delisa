// Cliente da API. Os caminhos são relativos: em desenvolvimento o Vite faz
// proxy de /api para o backend; em produção o nginx faz o mesmo.

class ApiError extends Error {
  constructor(message, status) {
    super(message)
    this.name = 'ApiError'
    this.status = status
  }
}

async function request(method, path, body) {
  let res
  try {
    res = await fetch(`/api${path}`, {
      method,
      headers: body === undefined ? undefined : { 'Content-Type': 'application/json' },
      body: body === undefined ? undefined : JSON.stringify(body),
    })
  } catch {
    throw new ApiError('Não foi possível falar com o servidor. Ele está rodando?', 0)
  }

  if (res.status === 204) return null

  const text = await res.text()
  let payload = null
  if (text) {
    try {
      payload = JSON.parse(text)
    } catch {
      payload = null
    }
  }

  if (!res.ok) {
    throw new ApiError(payload?.error || `Erro ${res.status} do servidor.`, res.status)
  }
  return payload
}

const get = (path) => request('GET', path)
const post = (path, body) => request('POST', path, body ?? {})
const put = (path, body) => request('PUT', path, body)
const del = (path) => request('DELETE', path)

export const api = {
  summary: () => get('/summary'),

  listClients: () => get('/clients'),
  createClient: (c) => post('/clients', c),
  updateClient: (id, c) => put(`/clients/${id}`, c),
  deleteClient: (id) => del(`/clients/${id}`),

  listServices: () => get('/services'),
  createService: (s) => post('/services', s),
  updateService: (id, s) => put(`/services/${id}`, s),
  deleteService: (id) => del(`/services/${id}`),

  listOrders: () => get('/orders'),
  createOrder: (o) => post('/orders', o),
  updateOrder: (id, o) => put(`/orders/${id}`, o),
  deleteOrder: (id) => del(`/orders/${id}`),

  listTransactions: () => get('/transactions'),
  createTransaction: (t) => post('/transactions', t),
  deleteTransaction: (id) => del(`/transactions/${id}`),

  exportBackup: () => get('/backup'),
  restoreBackup: (data) => post('/backup/restore', data),
  resetData: () => del('/data'),
}

export { ApiError }
