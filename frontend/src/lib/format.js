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
  return isoDate(d)
}

/** Formata uma Date como "2026-03-15", no fuso de quem está usando. Passar
    pelo toISOString() daria o dia em UTC, que à noite já é o dia seguinte. */
export function isoDate(d) {
  const pad = (n) => String(n).padStart(2, '0')
  return `${d.getFullYear()}-${pad(d.getMonth() + 1)}-${pad(d.getDate())}`
}

/** A data ISO somada de `dias`: addDays('2026-03-15', 2) -> "2026-03-17". */
export function addDays(iso, dias) {
  const d = new Date(`${iso}T00:00:00`)
  if (Number.isNaN(d.getTime())) return iso
  d.setDate(d.getDate() + dias)
  return isoDate(d)
}

/** Mostra o horário guardado: "14:30", ou "—" quando não foi combinado. */
export function fmtTime(hhmm) {
  return hhmm ? hhmm : '—'
}

/** Data e hora juntas, como aparecem na coluna de entrega: "15/03/2026 14:30". */
export function fmtDateTime(iso, hhmm) {
  const dia = fmtDate(iso)
  return hhmm && dia !== '—' ? `${dia} ${hhmm}` : dia
}

/** O mês de uma data ISO: "2026-03-15" -> "2026-03". */
export function monthOf(iso) {
  return String(iso || '').slice(0, 7)
}

/** Título do mês no calendário: "Março de 2026". */
export function fmtMonthTitle(mes) {
  const [ano, m] = String(mes).split('-').map(Number)
  const d = new Date(ano, m - 1, 1)
  if (Number.isNaN(d.getTime())) return '—'
  const s = d.toLocaleDateString('pt-BR', { month: 'long', year: 'numeric' })
  return s.charAt(0).toUpperCase() + s.slice(1)
}

/** O mês vizinho: shiftMonth('2026-03', -1) -> "2026-02". */
export function shiftMonth(mes, passos) {
  const [ano, m] = String(mes).split('-').map(Number)
  const d = new Date(ano, m - 1 + passos, 1)
  return monthOf(isoDate(d))
}

/**
 * As semanas do mês, como o calendário as desenha: cada semana tem os sete
 * dias, de domingo a sábado, e os dias do mês vizinho que completam a
 * primeira e a última linha vêm marcados com noMes: false.
 */
export function monthGrid(mes) {
  const [ano, m] = String(mes).split('-').map(Number)
  const primeiro = new Date(ano, m - 1, 1)
  if (Number.isNaN(primeiro.getTime())) return []

  const inicio = new Date(primeiro)
  inicio.setDate(1 - primeiro.getDay()) // recua até o domingo da primeira semana

  const semanas = []
  for (let s = 0; s < 6; s++) {
    const semana = []
    for (let d = 0; d < 7; d++) {
      const dia = new Date(inicio)
      dia.setDate(inicio.getDate() + s * 7 + d)
      semana.push({ iso: isoDate(dia), numero: dia.getDate(), noMes: dia.getMonth() === m - 1 })
    }
    semanas.push(semana)
  }
  // Fevereiro começando no domingo usa cinco linhas; a sexta sairia inteira
  // fora do mês.
  return semanas.filter((semana) => semana.some((c) => c.noMes))
}

export const DIAS_DA_SEMANA = ['Dom', 'Seg', 'Ter', 'Qua', 'Qui', 'Sex', 'Sáb']

/** Título do dia na agenda: "Hoje — sexta-feira, 15/03". */
export function fmtDayTitle(iso, hoje = today()) {
  const d = new Date(`${iso}T00:00:00`)
  if (Number.isNaN(d.getTime())) return 'Sem data de entrega'

  const semana = d.toLocaleDateString('pt-BR', { weekday: 'long' })
  const curto = d.toLocaleDateString('pt-BR', { day: '2-digit', month: '2-digit' })
  const titulo = `${semana.charAt(0).toUpperCase() + semana.slice(1)}, ${curto}`

  if (iso === hoje) return `Hoje — ${titulo}`
  if (iso === addDays(hoje, 1)) return `Amanhã — ${titulo}`
  return titulo
}

/**
 * Ordena pedidos pela hora combinada da entrega. Quem não tem hora marcada
 * cai no fim do dia, não na frente de quem tem horário.
 *
 * Fica aqui porque a tela da agenda e o quadro da segunda tela precisam
 * listar o dia na mesma ordem.
 */
export function porHora(a, b) {
  const ha = a.dueTime || '99:99'
  const hb = b.dueTime || '99:99'
  if (ha !== hb) return ha < hb ? -1 : 1
  return Number(a.number) - Number(b.number)
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

/* Etapas que não cobram atraso. "Pronto" entra aqui junto de entregue e
   cancelado: a peça está feita e o que falta é a cliente passar para buscar,
   o que não é trabalho atrasado do ateliê. */
const SEM_ATRASO = ['Pronto', 'Entregue', 'Cancelado']

/**
 * O pedido passou do dia combinado e ainda depende do ateliê.
 *
 * Mora aqui porque a tela da agenda e o quadro da segunda tela precisam
 * contar a mesma coisa como atrasada.
 *
 * Serve tanto para o pedido inteiro quanto para uma entrega de peça (ver
 * entregasDoPedido): as duas têm due, dueTime, status e items, e a conta é a
 * mesma.
 */
export function atrasado(order, hoje = today()) {
  if (!order?.due || order.due >= hoje) return false
  if (SEM_ATRASO.includes(order.status)) return false
  // Peça que a cliente já levou não é trabalho atrasado, mesmo que a etapa do
  // pedido ainda não tenha sido mudada: ela saiu do ateliê.
  return !todasRetiradas(order.items)
}

/* ---------- entrega e retirada por peça ----------

   A entrega era do pedido inteiro, e passou a poder ser de cada peça: a calça
   fica para sexta e o vestido para segunda. A data da peça é a exceção, e a
   do pedido continua sendo a combinação geral — peça sem data própria segue o
   pedido. É isso que faz o pedido comum, com tudo para o mesmo dia, continuar
   se comportando como antes em todas as telas. */

/**
 * A entrega combinada para uma peça: a dela, quando tem; senão a do pedido.
 * `propria` diz qual das duas é, para a tela poder mostrar a diferença.
 */
export function entregaDaPeca(order, itens) {
  // Os três campos são gravados iguais em todos os serviços da peça, então o
  // primeiro item responde pela peça toda.
  const primeiro = (itens || [])[0] || {}
  if (primeiro.due) {
    return { due: primeiro.due, dueTime: primeiro.dueTime || '', propria: true }
  }
  return { due: order?.due || '', dueTime: order?.dueTime || '', propria: false }
}

/** O dia em que a cliente levou a peça, ou '' se ela ainda está no ateliê. */
export function retiradaDaPeca(itens) {
  return ((itens || [])[0] || {}).deliveredAt || ''
}

/** Todas as peças já saíram do ateliê. Pedido sem item nenhum não conta. */
export function todasRetiradas(items) {
  const lista = items || []
  return lista.length > 0 && lista.every((i) => i.deliveredAt)
}

/** Quantas peças do pedido a cliente já levou, e quantas há no total. */
export function contagemDeRetiradas(order) {
  const grupos = groupItemsByPiece(order?.items)
  return {
    retiradas: grupos.filter((g) => retiradaDaPeca(g.itens)).length,
    total: grupos.length,
  }
}

/**
 * As entregas de um pedido: uma para cada data e hora combinadas, com as
 * peças que vencem nela.
 *
 * Cada entrega é uma cópia do pedido com a data, a hora e as peças daquele
 * compromisso — assim a agenda, que já sabia desenhar um pedido, desenha uma
 * entrega sem precisar aprender nada novo. `pedido` guarda o pedido de
 * verdade: é ele que vai para a tela de edição, porque a cópia tem só parte
 * dos serviços e salvá-la apagaria o resto.
 *
 * Pedido com tudo para o mesmo dia devolve uma entrega só, igual ao pedido
 * inteiro. O desdobramento aparece apenas quando ela marca a data de uma
 * peça.
 */
export function entregasDoPedido(order) {
  const porData = new Map()

  for (const grupo of groupItemsByPiece(order.items)) {
    const { due, dueTime, propria } = entregaDaPeca(order, grupo.itens)
    const chave = `${due}|${dueTime}`

    let entrega = porData.get(chave)
    if (!entrega) {
      entrega = {
        ...order,
        due,
        dueTime,
        items: [],
        pedido: order,
        chave: `${order.id}-${chave}`,
        pecasProprias: false,
      }
      porData.set(chave, entrega)
    }
    entrega.items.push(...grupo.itens)
    if (propria) entrega.pecasProprias = true
  }

  // Pedido sem item nenhum não existe na prática, mas se existir ele não pode
  // desaparecer da agenda por causa disso.
  if (porData.size === 0) {
    return [{ ...order, pedido: order, chave: `${order.id}-`, pecasProprias: false }]
  }

  const lista = [...porData.values()]
  // A entrega é "parcial" quando o pedido tem mais de uma: é o que a tela
  // avisa, para ela não achar que o pedido todo sai naquele dia.
  for (const e of lista) e.parcial = lista.length > 1
  return lista
}

/** Todas as entregas de todos os pedidos, numa lista só. */
export function todasAsEntregas(orders) {
  return (orders || []).flatMap(entregasDoPedido)
}

export const PAYMENT_METHODS = [
  'Não informado', 'Pix', 'Dinheiro', 'Cartão de débito', 'Cartão de crédito',
]

/* As formas que entram numa baixa. "Não informado" fica de fora: quando o
   dinheiro entra de verdade, ela sabe como entrou. */
export const FORMAS_DE_RECEBIMENTO = PAYMENT_METHODS.filter((p) => p !== 'Não informado')

/**
 * Quanto falta receber de um pedido, nunca negativo.
 *
 * O pedido pago a mais existe — a cliente arredondou para cima — e o saldo
 * negativo é guardado como está. Mas na hora de cobrar, o que falta é zero,
 * não um valor "a menos".
 */
export function faltaReceber(order) {
  return Math.max(0, Number(order?.balanceCents) || 0)
}

/** Pedido cancelado não deve nada: ninguém cobra o que não vai ser feito. */
export function temSaldoAberto(order) {
  return order?.status !== 'Cancelado' && faltaReceber(order) > 0
}

/**
 * Junta os serviços do pedido pela peça de roupa, mantendo a ordem em que
 * cada peça aparece: [{ peca, itens }]. Os serviços sem peça informada ficam
 * num grupo de peca vazia.
 *
 * É o que deixa a ficha, as listagens e o comprovante contarem a mesma
 * história: "na calça jeans, bainha e ajuste de cintura".
 */
export function groupItemsByPiece(items) {
  const grupos = []
  for (const item of items || []) {
    const peca = item.piece || ''
    let grupo = grupos.find((g) => g.peca === peca)
    if (!grupo) {
      grupo = { peca, itens: [] }
      grupos.push(grupo)
    }
    grupo.itens.push(item)
  }
  return grupos
}

/** Os serviços de um grupo em uma linha: "1× Bainha, 1× Ajuste de cintura". */
export function listItems(itens) {
  return (itens || []).map((i) => `${i.qty}× ${i.name}`).join(', ')
}

/** Quantas peças o pedido tem, somando as quantidades de cada serviço. */
export function orderQty(order) {
  return (order?.items || []).reduce((sum, i) => sum + (Number(i.qty) || 0), 0)
}

/* Entrega combinada para daqui a menos de 24 h é urgência. */
export const URGENCIA_HORAS = 24

/* Acréscimo sugerido quando o pedido é urgente. É só uma sugestão: a tela
   mostra o aviso e a Maria decide aplicar, mudar o número ou deixar zero. */
export const URGENCIA_PADRAO = 20

/* Hora assumida para a entrega de um pedido sem horário marcado: o fim do
   expediente do ateliê. Sem isso não haveria como medir o prazo de um pedido
   que só tem o dia combinado. */
const FIM_DO_EXPEDIENTE = '18:00'

/**
 * Quantas horas há entre a entrada do pedido e a entrega combinada, ou null
 * quando não há data de entrega para medir.
 *
 * A entrada do pedido é um dia, sem hora. Então: pedido que entra hoje conta
 * a partir de agora, que é o momento em que ela está fazendo a ficha; pedido
 * de outro dia conta do começo daquele dia.
 */
export function prazoEmHoras(date, due, dueTime, agora = new Date()) {
  if (!due) return null

  const alvo = new Date(`${due}T${dueTime || FIM_DO_EXPEDIENTE}:00`)
  if (Number.isNaN(alvo.getTime())) return null

  const base = !date || date === today() ? agora : new Date(`${date}T00:00:00`)
  if (Number.isNaN(base.getTime())) return null

  return (alvo.getTime() - base.getTime()) / 3_600_000
}

/**
 * Fecha a conta do pedido: subtotal dos serviços, desconto, acréscimo de
 * urgência e total.
 *
 * Os dois percentuais incidem sobre o subtotal, e cada um é arredondado uma
 * única vez — a mesma conta que a view order_totals faz no banco. Se as duas
 * divergirem, a tela mostra um total e o comprovante imprime outro.
 */
export function orderAmounts(items, discountPercent = 0, rushPercent = 0) {
  const itemsCents = (items || []).reduce(
    (sum, i) => sum + (Number(i.qty) || 0) * (Number(i.priceCents) || 0),
    0,
  )
  const parte = (pct) => Math.round((itemsCents * (Number(pct) || 0)) / 100)

  const discountCents = parte(discountPercent)
  const rushCents = parte(rushPercent)
  return {
    itemsCents,
    discountCents,
    rushCents,
    totalCents: itemsCents - discountCents + rushCents,
  }
}
