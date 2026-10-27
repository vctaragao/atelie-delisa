/* Aviso para a cliente pelo WhatsApp.

   Usa o link oficial de conversa (wa.me, o "click to chat" do próprio
   WhatsApp): o navegador abre a conversa com a mensagem já escrita e quem
   aperta enviar é a Maria. Não há cadastro, token, servidor nem custo por
   mensagem, e nada é enviado pelas costas dela.

   A alternativa seria a API oficial do WhatsApp Business, que exige conta na
   Meta, um número dedicado só para o sistema, modelos de mensagem aprovados
   antes de cada tipo de aviso e cobrança por conversa. Avisar que a peça
   ficou pronta cairia justamente na categoria que precisa de modelo aprovado.
   As bibliotecas não oficiais, que automatizam o WhatsApp Web por fora,
   contrariam os termos de uso e colocam o número do ateliê em risco de
   bloqueio — por isso não estão aqui. */

import { money, fmtDate, orderQty, groupItemsByPiece, listItems } from './format.js'

/**
 * O telefone da ficha no formato que o link do WhatsApp espera: só dígitos,
 * com o código do país na frente.
 *
 * Devolve null quando não dá para ter certeza do número — é melhor não abrir
 * conversa nenhuma do que abrir com o número de outra pessoa.
 */
export function numeroWhatsapp(phone) {
  const digitos = String(phone ?? '').replace(/\D/g, '')
  if (!digitos) return null

  // 11 dígitos = DDD + celular com o 9; 10 = DDD + fixo antigo. Falta o país.
  if (digitos.length === 10 || digitos.length === 11) return `55${digitos}`

  // Já veio com o 55 na frente.
  if (digitos.length >= 12 && digitos.length <= 13 && digitos.startsWith('55')) return digitos

  // Número de fora do Brasil, já com o código do país.
  if (digitos.length >= 12) return digitos

  // 8 ou 9 dígitos: é um número sem DDD, e adivinhar o DDD daria conversa com
  // estranho.
  return null
}

/** O link que abre a conversa com a mensagem pronta. */
export function linkWhatsapp(numero, texto) {
  return `https://wa.me/${numero}?text=${encodeURIComponent(texto)}`
}

/** "Ana Maria Souza" -> "Ana". Para a mensagem soar como ela falaria. */
function primeiroNome(nome) {
  const partes = String(nome ?? '').trim().split(/\s+/)
  return partes[0] || 'tudo bem'
}

/** Os serviços do pedido, agrupados pela peça de roupa:
    "calça jeans (1× Bainha, 1× Ajuste de cintura), 1× Troca de zíper". */
function listaDePecas(order) {
  return groupItemsByPiece(order.items)
    .map((g) => (g.peca ? `${g.peca} (${listItems(g.itens)})` : listItems(g.itens)))
    .join(', ')
}

function entregaCombinada(order) {
  if (!order.due) return ''
  const dia = fmtDate(order.due)
  return order.dueTime ? `${dia} às ${order.dueTime}` : dia
}

/* Os apelidos que podem aparecer no texto de uma mensagem. A tela de
   Mensagens mostra esta lista, e é por ela que a Maria sabe o que pode
   escrever entre chaves. */
export const APELIDOS = [
  { chave: '{cliente}', descricao: 'o primeiro nome da cliente' },
  { chave: '{cliente_completo}', descricao: 'o nome completo da cliente' },
  { chave: '{pedido}', descricao: 'o número do pedido' },
  { chave: '{servicos}', descricao: 'as peças e os serviços do pedido' },
  { chave: '{pecas}', descricao: 'quantas peças o pedido tem' },
  { chave: '{total}', descricao: 'o valor total' },
  { chave: '{pagamento}', descricao: '"já pago" ou "a pagar na retirada"' },
  { chave: '{entrega}', descricao: 'a data e a hora da entrega' },
  { chave: '{situacao}', descricao: 'a situação do pedido' },
]

/**
 * Troca os apelidos do texto pelos dados do pedido.
 *
 * Apelido que não existe fica como está, escrito mesmo: é melhor a mensagem
 * mostrar "{clinte}" e ela ver o erro de digitação do que o aviso chegar na
 * cliente com um buraco no meio.
 */
export function preencherMensagem(texto, order) {
  const valores = {
    cliente: primeiroNome(order.clientName),
    cliente_completo: String(order.clientName ?? '').trim(),
    pedido: order.number ?? '',
    servicos: listaDePecas(order),
    pecas: String(orderQty(order)),
    total: money(order.totalCents),
    pagamento: situacaoDoPagamento(order),
    entrega: entregaCombinada(order) || 'a combinar',
    situacao: order.status ?? '',
  }

  return String(texto ?? '').replace(/\{([a-z_]+)\}/g, (inteiro, chave) =>
    chave in valores ? valores[chave] : inteiro,
  )
}

/**
 * O que escrever no lugar de {pagamento}: a cliente que já pagou tudo não
 * pode receber um aviso pedindo dinheiro, e a que deixou um sinal precisa
 * saber quanto ainda falta.
 */
function situacaoDoPagamento(order) {
  const falta = Math.max(0, Number(order.balanceCents) || 0)
  const pago = Number(order.paidCents) || 0
  if (falta === 0) return 'já pago'
  if (pago > 0) return `faltam ${money(falta)}`
  return 'a pagar na retirada'
}

/** A mensagem que já vem escolhida para a situação atual do pedido. */
export function mensagemSugerida(mensagens, order) {
  const lista = mensagens || []
  return lista.find((m) => m.suggestStatus && m.suggestStatus === order.status) || lista[0] || null
}

/* Um pedido de mentira, para a tela de Mensagens mostrar como o texto fica
   preenchido antes de a Maria usá-lo com uma cliente de verdade. */
export const PEDIDO_DE_EXEMPLO = {
  number: '0042',
  clientName: 'Ana Maria Souza',
  status: 'Pronto',
  paid: false,
  paidCents: 0,
  balanceCents: 11000,
  due: '2026-03-15',
  dueTime: '14:30',
  totalCents: 11000,
  items: [
    { id: 1, name: 'Bainha', piece: 'Calça jeans', qty: 1, priceCents: 2500 },
    { id: 2, name: 'Ajuste de cintura', piece: 'Calça jeans', qty: 1, priceCents: 3500 },
    { id: 3, name: 'Troca de zíper', piece: 'Vestido de festa', qty: 1, priceCents: 3500 },
  ],
}
