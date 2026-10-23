/* Montagem do comprovante da impressora térmica de 58 mm.

   A impressora é uma POS58 genérica. Como o fabricante não distribui driver,
   ela está instalada no Windows com o "Generic / Text Only", que não desenha
   nada: repassa o texto caractere a caractere e joga fora todo o resto. Foi
   por isso que o cupom montado em HTML saiu com os rótulos grudados nos
   valores — o alinhamento era de CSS, e CSS não sobrevive a esse driver.

   Aqui o cupom é montado como texto puro, com os espaços contados. Duas
   restrições vêm do aparelho e aparecem nas constantes abaixo:

   - 32 colunas por linha (384 pontos de largura, 12 pontos por caractere);
   - tabela de caracteres PC936, a chinesa, gravada de fábrica e sem como
     trocar pelo navegador. Acento cai nessa tabela e sai como ideograma,
     então o comprovante é emitido sem acento. */

import { money, fmtDate, fmtDateTime } from './format.js'

/** Largura útil da bobina, em caracteres. */
export const COLUNAS = 32

const SEPARADOR = '-'.repeat(COLUNAS)

/* Linhas de folga impressas depois do rodapé.

   A cabeça de impressão fica cerca de 2 cm atrás do serrilhado. O que ela
   acabou de imprimir ainda está dentro do mecanismo, e só sai quando o papel
   anda de novo — ou seja, na próxima impressão. Sem essa folga o rodapé de um
   comprovante aparece grudado no topo do comprovante seguinte.

   Cada linha avança cerca de 3 mm. Com 8 o rodapé saía folgado demais e
   sobrava bobina à toa; 5 dão 1,5 cm, que é o bastante para o aviso final
   passar do serrilhado. Abaixo disso o rodapé começa a ficar preso de novo e
   reaparece no topo do comprovante seguinte. */
const LINHAS_DE_FOLGA = 5

/* Cada linha de folga leva um espaço, e não vazio.

   O driver "Generic / Text Only" não imprime a página: ele imprime o texto
   que o navegador desenha nela. Linha sem nenhum caractere não gera desenho
   nenhum, então o navegador não manda nada e o papel não anda — foi por isso
   que passar de 3 para 8 linhas vazias não mudou coisa alguma. Com um espaço
   existe algo a desenhar, e a linha chega até a impressora. */
const LINHA_DE_FOLGA = ' '

/* Deixa só o que a impressora sabe imprimir: tira acento ("Situação" vira
   "Situacao"), troca o espaço fixo que o formatador de moeda usa por um
   espaço comum e o travessão por hífen. O que sobrar de fora da tabela ASCII
   vira "?", que é melhor do que um ideograma no meio do cupom.

   Todas as trocas mantêm o tamanho do texto, senão a conta dos espaços de
   alinhamento sairia errada. */
export function semAcento(texto) {
  return String(texto ?? '')
    .normalize('NFD')
    .replace(/[̀-ͯ]/g, '')
    .replace(/ /g, ' ')
    .replace(/[‒-―]/g, '-')
    .replace(/[^\x20-\x7e]/g, '?')
}

/** Quebra o texto em linhas de até `largura`, sem partir palavra no meio. */
function quebrar(texto, largura = COLUNAS) {
  const linhas = []
  let atual = ''

  const fecha = () => {
    // Palavra maior que a linha inteira (um e-mail, por exemplo) é partida
    // no limite, que é o único jeito de ela caber.
    while (atual.length > largura) {
      linhas.push(atual.slice(0, largura))
      atual = atual.slice(largura)
    }
  }

  for (const palavra of semAcento(texto).split(/\s+/).filter(Boolean)) {
    if (!atual) atual = palavra
    else if (atual.length + 1 + palavra.length <= largura) atual += ` ${palavra}`
    else {
      fecha()
      if (atual) linhas.push(atual)
      atual = palavra
    }
    fecha()
  }
  if (atual) linhas.push(atual)
  return linhas
}

/** Centraliza o texto na largura da bobina. */
function centro(texto) {
  return quebrar(texto).map(
    (linha) => ' '.repeat(Math.floor((COLUNAS - linha.length) / 2)) + linha,
  )
}

/** Rótulo à esquerda e valor encostado à direita, na mesma linha. Quando os
    dois não cabem juntos, o valor desce para a linha de baixo, ainda à
    direita — é o que mantém a coluna dos valores legível. */
function par(rotulo, valor) {
  const esq = semAcento(rotulo)
  const dir = semAcento(valor)

  if (esq.length + 1 + dir.length <= COLUNAS) {
    return [esq + ' '.repeat(COLUNAS - esq.length - dir.length) + dir]
  }
  return [
    esq,
    ...quebrar(dir).map((linha) => ' '.repeat(COLUNAS - linha.length) + linha),
  ]
}

/* Sem "R$" nas linhas dos itens: em 32 colunas cada caractere conta, e a
   linha do total já diz que são reais. */
const valor = (centavos) => semAcento(money(centavos)).replace('R$', '').trim()

/** Monta o comprovante do pedido como texto pronto para a impressora.
    `cliente` é a ficha do cliente, de onde saem telefone e endereço. */
export function cupom(pedido, cliente, emitidoEm = new Date()) {
  const linhas = []

  linhas.push(...centro('ATELIÊ DELISA'))
  linhas.push(...centro('Comprovante de pedido'))
  linhas.push(SEPARADOR)
  linhas.push(...centro(`PEDIDO #${pedido.number}`))
  linhas.push(SEPARADOR)

  linhas.push(...par('Cliente', pedido.clientName))
  if (cliente?.phone) linhas.push(...par('Telefone', cliente.phone))
  if (cliente?.address) linhas.push(...par('Endereço', cliente.address))
  linhas.push(...par('Entrada', fmtDate(pedido.date)))
  linhas.push(...par('Entrega', fmtDateTime(pedido.due, pedido.dueTime)))
  linhas.push(...par('Situação', pedido.status))
  linhas.push(SEPARADOR)

  for (const item of pedido.items || []) {
    linhas.push(...quebrar(item.name))
    // A conta do item recuada, para o nome da peça se destacar.
    linhas.push(
      ...par(`  ${item.qty} x ${valor(item.priceCents)}`, valor(item.qty * item.priceCents)),
    )
  }

  linhas.push(SEPARADOR)
  // O subtotal só aparece quando há desconto ou urgência: sem eles, repetir o
  // mesmo valor duas vezes seguidas só gastaria bobina e confundiria a leitura.
  if (pedido.discountCents || pedido.rushCents) {
    linhas.push(...par('Serviços', valor(pedido.itemsCents)))
    if (pedido.discountCents) {
      linhas.push(...par(`Desconto ${pedido.discountPercent}%`, `-${valor(pedido.discountCents)}`))
    }
    if (pedido.rushCents) {
      linhas.push(...par(`Urgência ${pedido.rushPercent}%`, `+${valor(pedido.rushCents)}`))
    }
  }
  linhas.push(...par('TOTAL', money(pedido.totalCents)))
  linhas.push(...par('Pagamento', pedido.payment))
  linhas.push(...par('Situação', pedido.paid ? 'PAGO' : 'PENDENTE'))

  if (pedido.notes) {
    linhas.push(SEPARADOR)
    linhas.push(...quebrar('Observações'))
    linhas.push(...quebrar(pedido.notes))
  }

  linhas.push(SEPARADOR)
  linhas.push(...centro(`Emitido em ${emitidoEm.toLocaleDateString('pt-BR')}`))
  linhas.push(...centro('Este comprovante registra o pedido acordado entre o ateliê e o cliente.'))
  linhas.push(...centro('NÃO É NOTA FISCAL.'))

  linhas.push(...Array(LINHAS_DE_FOLGA).fill(LINHA_DE_FOLGA))

  return linhas.join('\n')
}
