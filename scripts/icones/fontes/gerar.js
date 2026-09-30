const fs = require('fs')
const sharp = require('sharp')
// O pacote e ESM: no CommonJS a funcao vem em .default.
const pngToIco = require('png-to-ico').default || require('png-to-ico')

// Multi-resolucao: o Windows escolhe o tamanho conforme o contexto -- 16px na
// barra de tarefas, 48px na area de trabalho, 256px na visualizacao grande.
// Um .ico so com 256 fica borrado quando reduzido.
const tamanhos = [16, 32, 48, 64, 128, 256]
const LADO = 256
const RAIO = 52

const cantosArredondados = Buffer.from(
  `<svg xmlns="http://www.w3.org/2000/svg" width="${LADO}" height="${LADO}">
     <rect width="${LADO}" height="${LADO}" rx="${RAIO}" fill="#fff"/>
   </svg>`
)

// Faixa diagonal no canto inferior esquerdo, nas cores do selo de teste que
// aparece dentro do sistema. Um triangulo grande continua legivel a 16px,
// enquanto uma borda fina de 18px viraria pouco mais de um pixel.
const faixaDeTeste = Buffer.from(
  `<svg xmlns="http://www.w3.org/2000/svg" width="${LADO}" height="${LADO}">
     <defs>
       <clipPath id="borda">
         <rect width="${LADO}" height="${LADO}" rx="${RAIO}"/>
       </clipPath>
     </defs>
     <g clip-path="url(#borda)">
       <polygon points="0,150 106,256 0,256" fill="#e6a532"/>
       <polygon points="0,196 60,256 0,256" fill="#8a5a00"/>
     </g>
   </svg>`
)

async function logoBase() {
  return sharp('logo.jpg')
    .resize(LADO, LADO, { fit: 'cover', kernel: 'lanczos3' })
    .ensureAlpha()
    .composite([{ input: cantosArredondados, blend: 'dest-in' }])
    .png()
    .toBuffer()
}

async function gravar(nome, buffer) {
  const pngs = []
  for (const t of tamanhos) {
    const arquivo = `out/${nome}-${t}.png`
    await sharp(buffer).resize(t, t, { kernel: 'lanczos3' }).png().toFile(arquivo)
    pngs.push(arquivo)
  }
  const ico = await pngToIco(pngs)
  fs.writeFileSync(`out/atelie-delisa-${nome}.ico`, ico)
  console.log(`${nome}: ${tamanhos.length} resolucoes`)
}

async function main() {
  fs.mkdirSync('out', { recursive: true })
  const base = await logoBase()
  await gravar('prod', base)

  const dev = await sharp(base)
    .composite([{ input: faixaDeTeste, blend: 'over' }])
    .png()
    .toBuffer()
  await gravar('dev', dev)

  const chat = await sharp(fs.readFileSync('chat.svg'), { density: 600 })
    .resize(LADO, LADO)
    .png()
    .toBuffer()
  await gravar('chat', chat)
}

main().catch((e) => { console.error(e); process.exit(1) })
