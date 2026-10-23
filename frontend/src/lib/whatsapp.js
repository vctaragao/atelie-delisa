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

import { money, fmtDate, orderQty } from './format.js'

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

/** As peças do pedido em uma linha: "2× Bainha de calça, 1× Troca de zíper". */
function listaDePecas(order) {
  return (order.items || []).map((i) => `${i.qty}× ${i.name}`).join(', ')
}

function entregaCombinada(order) {
  if (!order.due) return ''
  const dia = fmtDate(order.due)
  return order.dueTime ? `${dia} às ${order.dueTime}` : dia
}

/* Os modelos de mensagem. O texto é só o ponto de partida: a tela mostra numa
   caixa editável, e ela ajusta antes de abrir o WhatsApp. */
export const MODELOS = [
  {
    id: 'pronto',
    label: 'Serviço pronto para retirada',
    texto: (order) => {
      const linhas = [
        `Oi, ${primeiroNome(order.clientName)}! Aqui é do Ateliê Delisa.`,
        '',
        `Seu pedido #${order.number} está pronto: ${listaDePecas(order)}.`,
        `Total: ${money(order.totalCents)}${order.paid ? ' (já pago).' : ' (a pagar na retirada).'}`,
        '',
        'Pode passar para retirar quando quiser. Qualquer dúvida, é só responder por aqui.',
      ]
      return linhas.join('\n')
    },
  },
  {
    id: 'prazo',
    label: 'Lembrete da data de entrega',
    texto: (order) => {
      const quando = entregaCombinada(order)
      return [
        `Oi, ${primeiroNome(order.clientName)}! Aqui é do Ateliê Delisa.`,
        '',
        `Passando para lembrar do seu pedido #${order.number} ` +
          `(${orderQty(order)} ${orderQty(order) === 1 ? 'peça' : 'peças'})` +
          (quando ? `, combinado para ${quando}.` : '.'),
        '',
        'Se precisar mudar o horário, me avise que a gente ajeita.',
      ].join('\n')
    },
  },
  {
    id: 'pagamento',
    label: 'Lembrete de pagamento',
    texto: (order) =>
      [
        `Oi, ${primeiroNome(order.clientName)}! Aqui é do Ateliê Delisa.`,
        '',
        `Sobre o pedido #${order.number}: ficou pendente o valor de ` +
          `${money(order.totalCents)}.`,
        '',
        'Pode pagar na retirada ou por Pix, como ficar melhor para você.',
      ].join('\n'),
  },
]

/** O modelo que faz mais sentido para a situação atual do pedido. */
export function modeloSugerido(order) {
  if (order.status === 'Pronto') return 'pronto'
  if (!order.paid && order.status === 'Entregue') return 'pagamento'
  return 'prazo'
}
