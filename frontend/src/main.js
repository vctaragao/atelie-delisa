import { mount } from 'svelte'
import './app.css'
import App from './App.svelte'
import PainelAgenda from './routes/PainelAgenda.svelte'

// A mesma página serve o sistema inteiro e o quadro da agenda, que a Maria
// abre em outra janela para deixar numa segunda tela. Não há roteador neste
// projeto, e o parâmetro na URL é o bastante para a janela nova montar só o
// quadro, sem barra lateral e sem cabeçalho.
const painel = new URLSearchParams(location.search).get('painel') === 'agenda'

export default mount(painel ? PainelAgenda : App, { target: document.getElementById('app') })
