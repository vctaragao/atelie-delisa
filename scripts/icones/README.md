# Ícones dos atalhos

Os três `.ico` desta pasta são os ícones dos atalhos da área de trabalho.
Cada um traz seis resoluções (16, 32, 48, 64, 128 e 256), porque o Windows
escolhe o tamanho conforme o contexto — 16px na barra de tarefas, 48px na
área de trabalho, 256px na visualização grande. Um `.ico` com apenas 256
ficaria borrado ao ser reduzido.

| Arquivo | Desenho |
|---|---|
| `atelie-delisa-prod.ico` | o logo do ateliê, com cantos arredondados |
| `atelie-delisa-dev.ico` | o mesmo logo, com faixa diagonal âmbar no canto inferior esquerdo |
| `atelie-delisa-chat.ico` | a marca do Claude Code sobre o verde-petróleo do logo |

A faixa do ícone de desenvolvimento é um triângulo grande, e não uma borda
ao redor: uma borda de 18px nos 256 viraria pouco mais de um pixel a 16px e
sumiria justamente na barra de tarefas, onde mais importa distinguir os dois
ambientes. As cores são as mesmas do selo de teste que aparece dentro do
sistema, então o ícone e a tela contam a mesma história.

## Como aplicar num atalho

Botão direito no atalho → Propriedades → **Alterar ícone** → Procurar → e
escolher o arquivo desta pasta.

O caminho fica gravado no atalho, então estes arquivos precisam continuar
onde estão. É por isso que moram no repositório, e não na área de trabalho
ou em Downloads.

## Como gerar de novo

Os fontes estão em `fontes/`. A geração roda num container Node descartável,
sem depender de nada instalado na máquina:

```powershell
cd scripts\icones\fontes
docker run --rm -v "${PWD}:/work" -w /work node:22 sh -c "npm ci && node gerar.js"
```

Os `.ico` saem em `fontes/out/` e precisam ser copiados para a pasta acima.
Verificado: partindo de uma cópia limpa, o comando acima reproduz os três
arquivos byte a byte iguais aos que estão versionados.

`package-lock.json` está versionado para que `npm ci` instale exatamente as
mesmas versões de `sharp` e `png-to-ico` que geraram os arquivos atuais.

## Limitação conhecida

O `logo.jpg` de origem tem **150×150**. As resoluções até 128 saem bem, mas a
de 256 é uma ampliação e fica um pouco macia na visualização grande do
Explorer. Se aparecer o logo em resolução maior, basta substituir o arquivo e
gerar de novo.
