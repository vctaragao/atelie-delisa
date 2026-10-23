// Cliente da API. Os caminhos são relativos: em desenvolvimento o Vite faz
// proxy de /api para o backend; em produção o nginx faz o mesmo.

class ApiError extends Error {
  constructor(message, status) {
    super(message)
    this.name = 'ApiError'
    this.status = status
  }
}

async function send(method, path, init) {
  let res
  try {
    res = await fetch(`/api${path}`, { method, ...init })
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

function request(method, path, body) {
  return send(
    method,
    path,
    body === undefined
      ? {}
      : { headers: { 'Content-Type': 'application/json' }, body: JSON.stringify(body) },
  )
}

// Envia o arquivo como ele é, sem passar por JSON nem por texto: é isso que
// preserva os acentos de uma planilha salva pelo Excel, que não grava em UTF-8
// — o backend é que descobre a codificação.
const postFile = (path, file) =>
  send('POST', path, { headers: { 'Content-Type': 'text/csv' }, body: file })

const get = (path) => request('GET', path)
const post = (path, body) => request('POST', path, body ?? {})
const put = (path, body) => request('PUT', path, body)
const del = (path) => request('DELETE', path)

export const api = {
  summary: () => get('/summary'),
  report: (from, to) => get(`/report?from=${from}&to=${to}`),

  listClients: () => get('/clients'),
  createClient: (c) => post('/clients', c),
  updateClient: (id, c) => put(`/clients/${id}`, c),
  deleteClient: (id) => del(`/clients/${id}`),
  importClients: (file, update) =>
    postFile(`/clients/import${update ? '?update=1' : ''}`, file),

  listServices: () => get('/services'),
  createService: (s) => post('/services', s),
  updateService: (id, s) => put(`/services/${id}`, s),
  deleteService: (id) => del(`/services/${id}`),
  importServices: (file, update) =>
    postFile(`/services/import${update ? '?update=1' : ''}`, file),

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
