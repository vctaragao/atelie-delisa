const brl = new Intl.NumberFormat('pt-BR', { style: 'currency', currency: 'BRL' })

/** Formata centavos como moeda: 2500 -> "R$ 25,00". */
export function money(cents) {
  return brl.format((Number(cents) || 0) / 100)
}

/** Converte o valor digitado em reais para centavos, arredondando. */
export function toCents(reais) {
  const n = Number(String(reais ?? '').replace(',', '.'))
  if (!Number.isFinite(n)) return 0
  return Math.round(n * 100)
}

/** Converte centavos para o número em reais usado nos inputs. */
export function fromCents(cents) {
  return (Number(cents) || 0) / 100
}

/** Formata "2026-03-15" como "15/03/2026". */
export function fmtDate(iso) {
  if (!iso) return '—'
  const d = new Date(`${iso}T00:00:00`)
  if (Number.isNaN(d.getTime())) return '—'
  return d.toLocaleDateString('pt-BR')
}

/** Data de hoje no formato aceito por <input type="date">. */
export function today() {
  const d = new Date()
  const pad = (n) => String(n).padStart(2, '0')
  return `${d.getFullYear()}-${pad(d.getMonth() + 1)}-${pad(d.getDate())}`
}

/** Classe CSS do selo de status. */
export function statusClass(status) {
  return {
    'Orçamento': 'pendente',
    'Aguardando': 'pendente',
    'Em produção': 'producao',
    'Pronto': 'pronto',
    'Entregue': 'entregue',
    'Cancelado': 'cancelado',
  }[status] || 'pendente'
}

export const ORDER_STATUSES = [
  'Orçamento', 'Aguardando', 'Em produção', 'Pronto', 'Entregue', 'Cancelado',
]

export const PAYMENT_METHODS = [
  'Não informado', 'Pix', 'Dinheiro', 'Cartão de débito', 'Cartão de crédito',
]

/** Soma os itens de um pedido, em centavos. */
export function orderTotal(order) {
  return (order?.items || []).reduce(
    (sum, i) => sum + (Number(i.qty) || 0) * (Number(i.priceCents) || 0),
    0,
  )
}
