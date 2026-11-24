<script module>
  /* Um número por campo de busca na tela. Os ids dos elementos da lista são
     ligados ao campo por aria-controls e aria-activedescendant, e um pedido
     tem vários serviços: sem o contador, todos os campos apontariam para o
     mesmo id e o leitor de tela leria a lista do primeiro. */
  let proximo = 0
</script>

<script>
  /* Busca de serviço da tabela de preços.

     A tabela do ateliê passa de setenta serviços, com nomes que começam
     parecidos ("Ajustar ou soltar...", "Bainha de..."). Numa lista suspensa
     comum isso vira rolagem até achar: ela sabe o que quer, mas não onde
     está. Aqui ela digita um pedaço do nome e a lista encurta.

     A busca ignora acento e maiúscula, e aceita as palavras em qualquer
     ordem: "calca bainha" acha "Bainha de calça à mão". É o jeito como ela
     fala do serviço, não como ele está escrito na tabela.

     No fim da lista ficam as duas saídas para o que não está na tabela:
     cadastrar o serviço na tabela de preços sem sair do pedido, ou cobrar
     este uma vez só, sem cadastrar. Quem desenha o cadastro é o modal do
     pedido, que tem a largura para isso — aqui só se pede por ele.

     A lista de achados fica no fluxo da página, empurrando o resto para
     baixo, e não flutuando sobre ele: o modal do pedido tem rolagem própria
     (overflow: auto), e uma lista flutuante seria cortada na borda de baixo
     justamente quando o campo está perto do fim do modal. */

  import { money } from './format.js'

  let { servicos, serviceId, onpick, onnovo } = $props()

  const id = `busca-servico-${proximo++}`

  // O que está digitado na busca. `buscando` separa dois estados do mesmo
  // campo: fora da busca ele mostra o serviço já escolhido; dentro, mostra o
  // que ela está digitando.
  let busca = $state('')
  let buscando = $state(false)
  let destacado = $state(0)
  let painel = $state(null)

  // O destaque mudou pela seta do teclado, e não pelo ponteiro do mouse. Só
  // nesse caso a lista rola atrás dele: rolar enquanto ela passa o mouse
  // moveria o item para debaixo do ponteiro, que destacaria o vizinho, que
  // rolaria de novo — a lista ficaria tremendo sob a mão dela.
  let porTeclado = $state(false)

  const escolhido = $derived(servicos.find((s) => s.id === serviceId) || null)
  const texto = $derived(buscando ? busca : escolhido?.name || '')
  const achados = $derived(buscando ? filtrar(servicos, busca) : [])

  /* O serviço escolhido vira o texto apagado do campo enquanto ela busca:
     some do campo para dar lugar ao que ela digita, mas continua à vista,
     para ela saber o que está trocando — e que não perdeu nada se desistir. */
  const dica = $derived(escolhido?.name || 'Buscar serviço — ex.: bainha calça')

  /* As linhas da lista, achados e saídas na mesma sequência. O teclado anda
     por este array e o Enter age pelo `tipo` da linha destacada: assim a
     conta do que é cada posição existe num lugar só. */
  const opcoes = $derived([
    ...achados.map((s) => ({ tipo: 'servico', servico: s, key: `s${s.id}` })),
    { tipo: 'cadastrar', key: 'cadastrar' },
    { tipo: 'livre', key: 'livre' },
  ])

  /** "Calça" -> "calca": o que a busca compara, sem acento nem maiúscula. */
  function semAcento(s) {
    return String(s ?? '')
      .normalize('NFD')
      .replace(/[̀-ͯ]/g, '')
      .toLowerCase()
  }

  /** Os serviços que combinam com o que foi digitado. Cada palavra precisa
      aparecer no nome ou na categoria, em qualquer ordem. Busca vazia devolve
      a tabela inteira, para ela poder só olhar a lista como antes. */
  function filtrar(lista, termo) {
    const palavras = semAcento(termo).split(/\s+/).filter(Boolean)
    if (palavras.length === 0) return lista

    return lista.filter((s) => {
      const alvo = `${semAcento(s.name)} ${semAcento(s.category)}`
      return palavras.every((p) => alvo.includes(p))
    })
  }

  /* Abre a lista com a busca em branco, mostrando a tabela inteira — ela
     pode querer só olhar, como fazia na lista suspensa.

     Não faz nada se a lista já está aberta: o campo chama isto também no
     clique, para a lista voltar quando ela clica num campo que já está com o
     foco, e um clique no meio do que ela digitou não pode apagar a busca. */
  function abrir() {
    if (buscando) return
    buscando = true
    busca = ''
    destacado = 0
    porTeclado = false
  }

  /** Sai da busca sem mudar nada: o campo volta a mostrar o que estava
      escolhido. */
  function desistir() {
    buscando = false
    busca = ''
    destacado = 0
  }

  /* Fecha a busca com a escolha feita, mas deixa o foco onde está: daí o Tab
     segue para a quantidade e o preço, que é a ordem em que ela preenche. Um
     blur aqui jogaria o foco no nada e o Tab recomeçaria do topo da tela. */
  function escolher(escolha) {
    onpick(escolha)
    desistir()
  }

  /* Pede o cadastro do serviço na tabela, já com o que ela digitou como nome:
     quem abre o formulário é o modal do pedido. A busca fecha antes, para o
     cadastro não aparecer debaixo da lista aberta. */
  function cadastrar() {
    const nome = busca.trim()
    desistir()
    onnovo(nome)
  }

  /* Destaque pelo ponteiro: marca o item sob o mouse e desliga a rolagem
     automática, que é só da seta do teclado. */
  function destacar(indice) {
    destacado = indice
    porTeclado = false
  }

  function digitou(valor) {
    busca = valor
    buscando = true
    // Cada letra nova reinicia o destaque no primeiro achado: é ele que o
    // Enter escolhe.
    destacado = 0
    porTeclado = false
  }

  function agir(opcao) {
    if (!opcao) return
    if (opcao.tipo === 'servico') escolher(opcao.servico.id)
    else if (opcao.tipo === 'cadastrar') cadastrar()
    else escolher(null)
  }

  function teclou(event) {
    if (event.key === 'ArrowDown' || event.key === 'ArrowUp') {
      // Seta para baixo com a lista fechada abre a lista, em vez de mexer no
      // destaque de uma lista que ela não está vendo.
      if (!buscando) {
        abrir()
      } else {
        const passo = event.key === 'ArrowDown' ? 1 : -1
        const total = opcoes.length
        destacado = (destacado + passo + total) % total
        porTeclado = true
      }
      event.preventDefault()
      return
    }

    if (event.key === 'Enter') {
      // Sem isto o Enter salvaria o pedido: o campo está dentro do
      // formulário do modal.
      event.preventDefault()
      if (buscando) agir(opcoes[destacado])
      return
    }

    if (event.key === 'Escape') {
      // O Esc fecha a busca, e só ela. Sem parar a propagação, o mesmo Esc
      // chegaria à janela e fecharia o modal do pedido inteiro, levando
      // embora o que ela já tinha preenchido.
      if (buscando) {
        event.preventDefault()
        event.stopPropagation()
        desistir()
      }
    }
  }

  /* Mantém o item destacado à vista: com a tabela inteira na lista, descer
     pela seta sairia da parte visível e o destaque desapareceria. A ordem dos
     filhos do painel é a das opções, então o índice do destaque é a posição
     do elemento. */
  $effect(() => {
    if (!buscando || !porTeclado) return
    painel?.children[destacado]?.scrollIntoView({ block: 'nearest' })
  })
</script>

<div class="busca-servico">
  <input
    type="text"
    role="combobox"
    aria-expanded={buscando}
    aria-controls={`${id}-achados`}
    aria-activedescendant={buscando ? `${id}-item-${destacado}` : undefined}
    aria-autocomplete="list"
    aria-label="Serviço da tabela de preços"
    placeholder={dica}
    value={texto}
    oninput={(e) => digitou(e.currentTarget.value)}
    onfocus={abrir}
    onclick={abrir}
    onblur={desistir}
    onkeydown={teclou}
  />

  {#if buscando}
    <!-- O mousedown no painel é cancelado para o campo não perder o foco
         antes do clique virar escolha: o blur fecharia a lista no meio do
         caminho e o clique cairia no vazio.

         O tabindex="-1" tira o painel da navegação por Tab de propósito:
         quem fica com o foco é o campo de busca, e é o aria-activedescendant
         dele que diz qual item está destacado — o padrão do combobox. -->
    <!-- svelte-ignore a11y_no_noninteractive_element_interactions -->
    <div
      bind:this={painel}
      class="achados"
      id={`${id}-achados`}
      role="listbox"
      aria-label="Serviços encontrados"
      tabindex="-1"
      onmousedown={(e) => e.preventDefault()}
    >
      {#each opcoes as o, i (o.key)}
        <button
          type="button"
          class="achado"
          class:destacado={i === destacado}
          class:achado-saida={o.tipo !== 'servico'}
          id={`${id}-item-${i}`}
          role="option"
          aria-selected={o.tipo === 'servico'
            ? o.servico.id === serviceId
            : o.tipo === 'livre' && serviceId === null}
          onmouseenter={() => destacar(i)}
          onclick={() => agir(o)}
        >
          {#if o.tipo === 'servico'}
            <span class="achado-nome">{o.servico.name}</span>
            <span class="achado-preco">{money(o.servico.priceCents)}</span>
          {:else if o.tipo === 'cadastrar'}
            <span class="achado-nome">
              {#if busca.trim()}
                + Cadastrar "{busca.trim()}" na tabela de preços
              {:else}
                + Cadastrar um serviço novo na tabela de preços
              {/if}
            </span>
          {:else}
            <span class="achado-nome">
              {#if achados.length === 0 && busca.trim()}
                Cobrar só neste pedido, sem cadastrar
              {:else}
                Serviço personalizado, fora da tabela
              {/if}
            </span>
          {/if}
        </button>
      {/each}
    </div>
  {/if}
</div>
